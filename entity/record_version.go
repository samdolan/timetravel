package entity

type RecordVersion struct {
	ID          int               `json:"id"`
	Version     int               `json:"version"`
	CreatedAtMS int64             `json:"created_at_ms"`
	Data        map[string]string `json:"data"`
}
