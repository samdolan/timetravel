package service

import (
	"context"
	"errors"

	"github.com/rainbowmga/timetravel/entity"
)

var ErrRecordDoesNotExist = errors.New("record with that id does not exist")
var ErrRecordIDInvalid = errors.New("record id must >= 0")
var ErrRecordAlreadyExists = errors.New("record already exists")
var ErrRecordVersionDoesNotExist = errors.New("record version does not exist")

// Implements method to get, create, and update record data.
type RecordService interface {

	// GetRecord will retrieve an record.
	GetRecord(ctx context.Context, id int) (entity.Record, error)

	// CreateRecord will insert a new record.
	//
	// If it a record with that id already exists it will fail.
	CreateRecord(ctx context.Context, record entity.Record) error

	// UpdateRecord will change the internal `Map` values of the record if they exist.
	// if the update[key] is null it will delete that key from the record's Map.
	//
	// UpdateRecord will error if id <= 0 or the record does not exist with that id.
	UpdateRecord(ctx context.Context, id int, updates map[string]*string) (entity.Record, error)
}

// VersionedRecordService adds access to historical versions of a record.
type VersionedRecordService interface {
	RecordService
	GetLatestRecordVersion(ctx context.Context, id int) (entity.RecordVersion, error)
	GetRecordVersion(ctx context.Context, id int, version int) (entity.RecordVersion, error)
}
