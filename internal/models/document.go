package models

import "time"

type Document struct {
	ID        string    `json:"id"`
	OwnerID   string    `json:"owner_id"`
	Name      string    `json:"name"`
	Mime      string    `json:"mime"`
	IsFile    bool      `json:"is_file"`
	IsPublic  bool      `json:"is_public"`
	Path      string    `json:"path"`
	JSONData  []byte    `json:"json_data"`
	Grants    []string  `json:"grants"`
	CreatedAt time.Time `json:"created_at"`
}

type DocumentFilter struct {
	Key   string
	Value string
	Limit int
}

var allowedKeys = map[string]bool{
	"name": true,
	"mime": true,
}

func (f DocumentFilter) IsValid() bool {
	if f.Key == "" && f.Value != "" {
		return false
	}
	if f.Key != "" && !allowedKeys[f.Key] {
		return false
	}
	return true
}
