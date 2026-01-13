package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rainbowmga/timetravel/entity"
	"github.com/rainbowmga/timetravel/service"
)

// GET /records/{id}
func (a *V2API) GetRecordLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	idNumber, err := strconv.ParseInt(id, 10, 32)
	if err != nil || idNumber <= 0 {
		err := writeError(w, "invalid id; id must be a positive number", http.StatusBadRequest)
		logError(err)
		return
	}

	at := r.URL.Query().Get("at")
	var recordVersion entity.RecordVersion
	if at == "" {
		recordVersion, err = a.records.GetLatestRecordVersion(ctx, int(idNumber))
	} else {
		atTime, parseErr := time.Parse(time.RFC3339Nano, at)
		if parseErr != nil {
			err := writeError(w, "invalid at; must be an RFC3339 timestamp", http.StatusBadRequest)
			logError(err)
			return
		}
		recordVersion, err = a.records.GetRecordVersionAt(ctx, int(idNumber), atTime.UTC().UnixMilli())
	}
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := ErrInternal.Error()
		if errors.Is(err, service.ErrRecordDoesNotExist) {
			statusCode = http.StatusBadRequest
			message = "record does not exist"
		}

		err := writeError(w, message, statusCode)
		logError(err)
		return
	}

	err = writeJSON(w, recordVersion, http.StatusOK)
	logError(err)
}
