package storage

import (
	"fileserver/internal/models"
	"io"
)

type FileRepository interface {
	SaveFile(doc *models.Document, reader io.Reader) error
	LoadFile(doc *models.Document) (io.ReadCloser, error)
	DeleteFile(doc *models.Document) error
}
