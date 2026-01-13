package entity

type Record struct {
	ID   int               `json:"id"`
	Data map[string]string `json:"data"`
}
