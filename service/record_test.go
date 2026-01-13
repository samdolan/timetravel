package service

import (
	"context"
	"testing"

	"github.com/rainbowmga/timetravel/entity"
)

func TestInMemoryRecordService_CreateGetUpdateAndCopyIsolation(t *testing.T) {
	ctx := context.Background()
	svc := NewInMemoryRecordService()

	if err := svc.CreateRecord(ctx, entity.Record{ID: 1, Data: map[string]string{"hello": "world"}}); err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}

	got, err := svc.GetRecord(ctx, 1)
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	if got.ID != 1 || got.Data["hello"] != "world" {
		t.Fatalf("unexpected record: %+v", got)
	}

	updated, err := svc.UpdateRecord(ctx, 1, map[string]*string{
		"hello":  ptr("world 2"),
		"status": ptr("ok"),
	})
	if err != nil {
		t.Fatalf("UpdateRecord: %v", err)
	}
	if updated.Data["hello"] != "world 2" || updated.Data["status"] != "ok" {
		t.Fatalf("unexpected updated record: %+v", updated)
	}

	updated2, err := svc.UpdateRecord(ctx, 1, map[string]*string{
		"hello": nil,
	})
	if err != nil {
		t.Fatalf("UpdateRecord (delete): %v", err)
	}
	if _, ok := updated2.Data["hello"]; ok {
		t.Fatalf("expected key to be deleted, got %+v", updated2.Data)
	}
	if updated2.Data["status"] != "ok" {
		t.Fatalf("expected other keys preserved, got %+v", updated2.Data)
	}
}

func ptr(s string) *string { return &s }
