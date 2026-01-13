package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/rainbowmga/timetravel/entity"
)

type DBRecordService struct {
	db *sql.DB
}

func NewDBRecordService(dbPath string) (*DBRecordService, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("dbPath is required")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS records (
			id INTEGER PRIMARY KEY,
			data_json TEXT NOT NULL
		)
	`); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &DBRecordService{db: db}, nil
}

func (s *DBRecordService) Close() error {
	return s.db.Close()
}

func (s *DBRecordService) GetRecord(ctx context.Context, id int) (entity.Record, error) {
	if id <= 0 {
		return entity.Record{}, ErrRecordIDInvalid
	}

	var dataJSON string
	err := s.db.QueryRowContext(ctx, `SELECT data_json FROM records WHERE id = ?`, id).Scan(&dataJSON)
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

	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO records (id, data_json) VALUES (?, ?)`,
		record.ID,
		string(dataJSONBytes),
	)
	if err != nil {
		// Keep behavior consistent with the in-memory implementation.
		if strings.Contains(strings.ToLower(err.Error()), "constraint") {
			return ErrRecordAlreadyExists
		}
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("expected 1 row affected, got %d", rows)
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

	var dataJSON string
	err = tx.QueryRowContext(ctx, `SELECT data_json FROM records WHERE id = ?`, id).Scan(&dataJSON)
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
	// Handle PATCH functionality
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

	if _, err := tx.ExecContext(ctx, `UPDATE records SET data_json = ? WHERE id = ?`, string(newDataJSONBytes), id); err != nil {
		return entity.Record{}, err
	}
	if err := tx.Commit(); err != nil {
		return entity.Record{}, err
	}

	return entity.Record{ID: id, Data: data}, nil
}
