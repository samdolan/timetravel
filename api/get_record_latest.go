package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
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

	recordVersion, err := a.records.GetLatestRecordVersion(ctx, int(idNumber))
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
