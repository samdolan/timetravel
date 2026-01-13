package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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
	return router
}
