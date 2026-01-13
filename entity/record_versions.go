package entity

type RecordVersions struct {
	ID       int                 `json:"id"`
	Versions []RecordVersionInfo `json:"versions"`
}

type RecordVersionInfo struct {
	Version     int               `json:"version"`
	CreatedAtMS int64             `json:"created_at_ms"`
	Data        map[string]string `json:"data"`
}
