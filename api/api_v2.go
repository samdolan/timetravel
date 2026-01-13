package api

import (
	"github.com/gorilla/mux"
	"github.com/rainbowmga/timetravel/service"
)

type V2API struct {
	records service.VersionedRecordService
}

func NewV2API(records service.VersionedRecordService) *V2API {
	return &V2API{records: records}
}

func (a *V2API) CreateRoutes(routes *mux.Router) {
	routes.Path("/records/{id}").HandlerFunc(a.GetRecordLatest).Methods("GET")
	routes.Path("/records/{id}/versions/{version}").HandlerFunc(a.GetRecordVersion).Methods("GET")
}
