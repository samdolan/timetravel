package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mattn/go-sqlite3"

	"github.com/rainbowmga/timetravel/entity"
)

type DBRecordService struct {
	db *sql.DB
}

func NewDBRecordService(dbPath string) (*DBRecordService, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("dbPath is required")
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_txlock=immediate", dbPath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DBRecordService{db: db}, nil
}

func (s *DBRecordService) Close() error {
	return s.db.Close()
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS record_versions (
			record_id     INTEGER NOT NULL,
			version       INTEGER NOT NULL,
			data_json     TEXT NOT NULL,
			created_at_ms INTEGER NOT NULL DEFAULT (unixepoch('now') * 1000),
			PRIMARY KEY (record_id, version)
		)
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_record_versions_created_at_ms ON record_versions (created_at_ms)`); err != nil {
		return err
	}

	// Supports queries that filter by version across all records.
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_record_versions_version ON record_versions (version)`); err != nil {
		return err
	}

	// Migration from the earlier Objective #1 schema (single-row `records` table).
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO record_versions (record_id, version, data_json)
		SELECT id, 1, data_json
		FROM records
	`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return err
	}

	return nil
}

func hasColumn(db *sql.DB, tableName, columnName string) (bool, error) {
	var marker int
	query := fmt.Sprintf(`SELECT 1 FROM pragma_table_info('%s') WHERE name = ? LIMIT 1`, tableName)
	err := db.QueryRow(query, columnName).Scan(&marker)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func (s *DBRecordService) GetRecord(ctx context.Context, id int) (entity.Record, error) {
	if id <= 0 {
		return entity.Record{}, ErrRecordIDInvalid
	}

	var dataJSON string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT data_json FROM record_versions WHERE record_id = ? ORDER BY version DESC LIMIT 1`,
		id,
	).Scan(&dataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Record{}, ErrRecordDoesNotExist
		}
		return entity.Record{}, err
	}

	var data map[string]string
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return entity.Record{}, err
	}
	if data == nil {
		data = map[string]string{}
	}

	return entity.Record{ID: id, Data: data}, nil
}

func (s *DBRecordService) GetLatestRecordVersion(ctx context.Context, id int) (entity.RecordVersion, error) {
	if id <= 0 {
		return entity.RecordVersion{}, ErrRecordIDInvalid
	}

	var (
		version     int
		dataJSON    string
		createdAtMS int64
	)
	err := s.db.QueryRowContext(
		ctx,
		`SELECT version, data_json, created_at_ms FROM record_versions WHERE record_id = ? ORDER BY version DESC LIMIT 1`,
		id,
	).Scan(&version, &dataJSON, &createdAtMS)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.RecordVersion{}, ErrRecordDoesNotExist
		}
		return entity.RecordVersion{}, err
	}

	var data map[string]string
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return entity.RecordVersion{}, err
	}
	if data == nil {
		data = map[string]string{}
	}

	return entity.RecordVersion{ID: id, Version: version, CreatedAtMS: createdAtMS, Data: data}, nil
}

func (s *DBRecordService) GetRecordVersion(ctx context.Context, id int, version int) (entity.RecordVersion, error) {
	if id <= 0 {
		return entity.RecordVersion{}, ErrRecordIDInvalid
	}
	if version <= 0 {
		return entity.RecordVersion{}, ErrRecordIDInvalid
	}

	var dataJSON string
	var createdAtMS int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT data_json, created_at_ms FROM record_versions WHERE record_id = ? AND version = ? LIMIT 1`,
		id,
		version,
	).Scan(&dataJSON, &createdAtMS)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.RecordVersion{}, ErrRecordVersionDoesNotExist
		}
		return entity.RecordVersion{}, err
	}

	var data map[string]string
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return entity.RecordVersion{}, err
	}
	if data == nil {
		data = map[string]string{}
	}

	return entity.RecordVersion{ID: id, Version: version, CreatedAtMS: createdAtMS, Data: data}, nil
}

func (s *DBRecordService) CreateRecord(ctx context.Context, record entity.Record) error {
	if record.ID <= 0 {
		return ErrRecordIDInvalid
	}

	data := record.Data
	if data == nil {
		data = map[string]string{}
	}
	dataJSONBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO record_versions (record_id, version, data_json) VALUES (?, 1, ?)`,
		record.ID,
		string(dataJSONBytes),
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return ErrRecordAlreadyExists
		}
		return err
	}
	return nil
}

func (s *DBRecordService) UpdateRecord(ctx context.Context, id int, updates map[string]*string) (entity.Record, error) {
	if id <= 0 {
		return entity.Record{}, ErrRecordIDInvalid
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return entity.Record{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var currentVersion int
	var dataJSON string
	err = tx.QueryRowContext(
		ctx,
		`SELECT version, data_json FROM record_versions WHERE record_id = ? ORDER BY version DESC LIMIT 1`,
		id,
	).Scan(&currentVersion, &dataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return entity.Record{}, ErrRecordDoesNotExist
		}
		return entity.Record{}, err
	}

	var data map[string]string
	if err := json.Unmarshal([]byte(dataJSON), &data); err != nil {
		return entity.Record{}, err
	}
	if data == nil {
		data = map[string]string{}
	}
	for key, value := range updates {
		if value == nil {
			delete(data, key)
		} else {
			data[key] = *value
		}
	}

	newDataJSONBytes, err := json.Marshal(data)
	if err != nil {
		return entity.Record{}, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO record_versions (record_id, version, data_json) VALUES (?, ?, ?)`,
		id,
		currentVersion+1,
		string(newDataJSONBytes),
	); err != nil {
		return entity.Record{}, err
	}
	if err := tx.Commit(); err != nil {
		return entity.Record{}, err
	}

	return entity.Record{ID: id, Data: data}, nil
}
