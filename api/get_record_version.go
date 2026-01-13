package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/rainbowmga/timetravel/service"
)

// GET /records/{id}/versions/{version}
func (a *V2API) GetRecordVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)

	idNumber, err := strconv.ParseInt(vars["id"], 10, 32)
	if err != nil || idNumber <= 0 {
		err := writeError(w, "invalid id; id must be a positive number", http.StatusBadRequest)
		logError(err)
		return
	}

	versionNumber, err := strconv.ParseInt(vars["version"], 10, 32)
	if err != nil || versionNumber <= 0 {
		err := writeError(w, "invalid version; version must be a positive number", http.StatusBadRequest)
		logError(err)
		return
	}

	recordVersion, err := a.records.GetRecordVersion(ctx, int(idNumber), int(versionNumber))
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := ErrInternal.Error()
		if errors.Is(err, service.ErrRecordVersionDoesNotExist) || errors.Is(err, service.ErrRecordDoesNotExist) {
			statusCode = http.StatusBadRequest
			message = "record/version does not exist"
		}

		err := writeError(w, message, statusCode)
		logError(err)
		return
	}

	err = writeJSON(w, recordVersion, http.StatusOK)
	logError(err)
}
