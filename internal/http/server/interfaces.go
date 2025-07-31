package server

import (
	"context"
	"fileserver/internal/models"
	"io"
)

type AuthService interface {
	Register(ctx context.Context, login string, password string, token string) (string, error)
	Login(ctx context.Context, login string, password string) (string, error)
	UserByToken(ctx context.Context, token string) (*models.User, error)
	Logout(ctx context.Context, token string) error
}

type DocumentService interface {
	UploadDocument(ctx context.Context, requester *models.User, doc *models.Document, content io.Reader) (string, error)
	DocumentByID(ctx context.Context, docID string, requester *models.User) (*models.Document, io.ReadCloser, error)
	DeleteDocument(ctx context.Context, docID string, requester *models.User) error
	GrantAccess(ctx context.Context, docID string, requester *models.User, grants []string) error
	ListDocuments(ctx context.Context, requester *models.User, login string, filter models.DocumentFilter) ([]*models.Document, error)
}

type UserService interface {
	UserIDByLogin(ctx context.Context, login string) (string, error)
}

type SessionStorer interface {
	UserByToken(ctx context.Context, login string) (*models.User, error)
}
