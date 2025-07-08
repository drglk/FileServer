package documentservice

import (
	"context"
	"fileserver/internal/models"
	"io"
)

type DocumentRepository interface {
	CreateDocument(ctx context.Context, doc *models.Document) error
	Delete(ctx context.Context, id string) error
	DocumentByID(ctx context.Context, id string) (*models.Document, error)
	DocumentsGrantedTo(ctx context.Context, userID string) ([]*models.Document, error)
	GrantAccess(ctx context.Context, docID string, userIDs []string) error
	ListByUser(ctx context.Context, userID string) ([]*models.Document, error)
	FilteredDocuments(ctx context.Context, login string, requesterID string, filter models.DocumentFilter) ([]*models.Document, error)
}

type FileStorage interface {
	SaveFile(doc *models.Document, reader io.Reader) error
	LoadFile(doc *models.Document) (io.ReadCloser, error)
	DeleteFile(doc *models.Document) error
}

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}) error
	Del(ctx context.Context, keys ...string) error
}
