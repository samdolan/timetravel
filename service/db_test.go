package service

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rainbowmga/timetravel/entity"
)

func TestDBRecordService_PersistsAcrossRestarts(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "timetravel.db")

	svc1, err := NewDBRecordService(dbPath)
	if err != nil {
		t.Fatalf("NewDBRecordService: %v", err)
	}

	if err := svc1.CreateRecord(ctx, entity.Record{ID: 1, Data: map[string]string{"hello": "world"}}); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if err := svc1.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	svc2, err := NewDBRecordService(dbPath)
	if err != nil {
		t.Fatalf("NewDBRecordService (reopen): %v", err)
	}
	t.Cleanup(func() { _ = svc2.Close() })

	got, err := svc2.GetRecord(ctx, 1)
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	if got.ID != 1 || got.Data["hello"] != "world" {
		t.Fatalf("unexpected record: %+v", got)
	}
}
