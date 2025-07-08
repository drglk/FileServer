package dto

import (
	"mime/multipart"
	"time"
)

type UploadDocumentRequest struct {
	Meta UploadMeta
	JSON []byte
	File multipart.File
}

type UploadMeta struct {
	Name     string   `json:"name"`
	IsFile   bool     `json:"file"`
	IsPublic bool     `json:"public"`
	Token    string   `json:"token"`
	Mime     string   `json:"mime"`
	Grants   []string `json:"grant"`
}

type DocumentResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mime      string    `json:"mime"`
	IsFile    bool      `json:"file"`
	IsPublic  bool      `json:"public"`
	CreatedAt time.Time `json:"created"`
	Grants    []string  `json:"grants"`
}
