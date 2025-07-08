package docs

import (
	"context"
	"fileserver/internal/models"
	"io"
)

const pkg = "docsHandler/"

type DocumentUploader interface {
	UploadDocument(ctx context.Context, user *models.User, doc *models.Document, content io.Reader) (string, error)
}

type DocumentProvider interface {
	ListDocuments(ctx context.Context, requester *models.User, login string, filter models.DocumentFilter) ([]*models.Document, error)
	DocumentByID(ctx context.Context, docID string, requester *models.User) (*models.Document, io.ReadCloser, error)
}

type DocumentDeleter interface {
	DeleteDocument(ctx context.Context, docID string, requester *models.User) error
}

type UserIDProvider interface {
	UserIDByLogin(ctx context.Context, login string) (string, error)
}

type AuthService interface {
	UserByToken(ctx context.Context, token string) (*models.User, error)
}
