package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/rainbowmga/timetravel/api"
	"github.com/rainbowmga/timetravel/entity"
	"github.com/rainbowmga/timetravel/service"
)

func TestV1_Health(t *testing.T) {
	router := newV1Router(t)

	rr := doRequest(router, http.MethodPost, "/api/v1/health", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var body map[string]bool
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["ok"] != true {
		t.Fatalf("expected ok=true, got %v", body)
	}
}

func TestV1_Records_Lifecycle(t *testing.T) {
	router := newV1Router(t)

	// Create.
	rr := doRequest(router, http.MethodPost, "/api/v1/records/1", `{"hello":"world"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Get.
	rr = doRequest(router, http.MethodGet, "/api/v1/records/1", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got entity.Record
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal get: %v", err)
	}
	if got.ID != 1 || got.Data["hello"] != "world" {
		t.Fatalf("unexpected record: %+v", got)
	}

	// Update.
	rr = doRequest(router, http.MethodPost, "/api/v1/records/1", `{"hello":"world 2","status":"ok"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", rr.Code, rr.Body.String())
	}

	// Delete a field.
	rr = doRequest(router, http.MethodPost, "/api/v1/records/1", `{"hello":null}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete status=%d body=%s", rr.Code, rr.Body.String())
	}
	var deleted entity.Record
	if err := json.Unmarshal(rr.Body.Bytes(), &deleted); err != nil {
		t.Fatalf("unmarshal delete: %v", err)
	}
	if deleted.ID != 1 {
		t.Fatalf("unexpected record: %+v", deleted)
	}
	if _, ok := deleted.Data["hello"]; ok {
		t.Fatalf("expected hello to be deleted, got %+v", deleted)
	}
	if deleted.Data["status"] != "ok" {
		t.Fatalf("expected status to remain, got %+v", deleted)
	}
}

func TestV1_Records_InvalidID(t *testing.T) {
	router := newV1Router(t)
	rr := doRequest(router, http.MethodGet, "/api/v1/records/0", "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestV1_Records_MissingRecord(t *testing.T) {
	router := newV1Router(t)
	rr := doRequest(router, http.MethodGet, "/api/v1/records/32", "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestV1_Records_InvalidJSON(t *testing.T) {
	router := newV1Router(t)
	rr := doRequest(router, http.MethodPost, "/api/v1/records/1", "{")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestV2_Records_GetSpecificVersion(t *testing.T) {
	router := newV1V2Router(t)

	_ = doRequest(router, http.MethodPost, "/api/v1/records/1", `{"hello":"world"}`)
	_ = doRequest(router, http.MethodPost, "/api/v1/records/1", `{"hello":"world 2","status":"ok"}`)

	at := doRequest(router, http.MethodGet, "/api/v2/records/1?at=1970-01-01T00:00:00Z", "")
	if at.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", at.Code, at.Body.String())
	}

	versions := doRequest(router, http.MethodGet, "/api/v2/records/1/versions", "")
	if versions.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", versions.Code, versions.Body.String())
	}
	var list entity.RecordVersions
	if err := json.Unmarshal(versions.Body.Bytes(), &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if list.ID != 1 || len(list.Versions) != 2 || list.Versions[0].Version != 1 || list.Versions[1].Version != 2 {
		t.Fatalf("unexpected versions: %+v", list)
	}
	if list.Versions[0].CreatedAtMS == 0 || list.Versions[1].CreatedAtMS == 0 {
		t.Fatalf("expected created_at_ms to be set: %+v", list)
	}
	if list.Versions[0].Data["hello"] != "world" {
		t.Fatalf("unexpected v1 data: %+v", list.Versions[0])
	}
	if list.Versions[1].Data["hello"] != "world 2" || list.Versions[1].Data["status"] != "ok" {
		t.Fatalf("unexpected v2 data: %+v", list.Versions[1])
	}

	latest := doRequest(router, http.MethodGet, "/api/v2/records/1", "")
	if latest.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", latest.Code, latest.Body.String())
	}
	var latestRecord entity.RecordVersion
	if err := json.Unmarshal(latest.Body.Bytes(), &latestRecord); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if latestRecord.ID != 1 || latestRecord.Version != 2 || latestRecord.Data["hello"] != "world 2" {
		t.Fatalf("unexpected record: %+v", latestRecord)
	}

	// Use the created_at_ms from version 1 to time-travel back to it.
	v1At := time.UnixMilli(list.Versions[0].CreatedAtMS).UTC().Format(time.RFC3339Nano)
	at = doRequest(router, http.MethodGet, "/api/v2/records/1?at="+url.QueryEscape(v1At), "")
	if at.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", at.Code, at.Body.String())
	}
	var atRecord entity.RecordVersion
	if err := json.Unmarshal(at.Body.Bytes(), &atRecord); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if atRecord.ID != 1 || atRecord.Version != 1 || atRecord.Data["hello"] != "world" {
		t.Fatalf("unexpected record: %+v", atRecord)
	}

	rr := doRequest(router, http.MethodGet, "/api/v2/records/1/versions/1", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var record entity.RecordVersion
	if err := json.Unmarshal(rr.Body.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record.ID != 1 || record.Version != 1 || record.Data["hello"] != "world" {
		t.Fatalf("unexpected record: %+v", record)
	}

	rr = doRequest(router, http.MethodGet, "/api/v2/records/1/versions/2", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record.ID != 1 || record.Version != 2 || record.Data["hello"] != "world 2" || record.Data["status"] != "ok" {
		t.Fatalf("unexpected record: %+v", record)
	}
}

func doRequest(router http.Handler, method, path, body string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rr, req)
	return rr
}

func newV1Router(t *testing.T) *mux.Router {
	return newTestRouter(t, false)
}

func newV1V2Router(t *testing.T) *mux.Router {
	return newTestRouter(t, true)
}

func newTestRouter(t *testing.T, includeV2 bool) *mux.Router {
	t.Helper()

	router := mux.NewRouter()
	dbPath := filepath.Join(t.TempDir(), "timetravel.db")
	recordService, err := service.NewDBRecordService(dbPath)
	if err != nil {
		t.Fatalf("NewDBRecordService: %v", err)
	}
	t.Cleanup(func() { _ = recordService.Close() })

	a := api.NewAPI(recordService)

	apiRoute := router.PathPrefix("/api/v1").Subrouter()
	apiRoute.Path("/health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})
	a.CreateRoutes(apiRoute)

	if includeV2 {
		v2 := api.NewV2API(recordService)
		v2Route := router.PathPrefix("/api/v2").Subrouter()
		v2.CreateRoutes(v2Route)
	}

	return router
}
