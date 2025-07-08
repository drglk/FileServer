package authservice

import (
	"context"
	"fileserver/internal/models"
)

type UserAdder interface {
	AddUser(ctx context.Context, user models.User) error
}

type UserProvider interface {
	UserByID(ctx context.Context, id string) (*models.User, error)
	UserByLogin(ctx context.Context, login string) (*models.User, error)
}

type SessionStorer interface {
	SaveSession(ctx context.Context, token string, userJSON string) error
	DeleteSession(ctx context.Context, token string) error
	GetUserByToken(ctx context.Context, token string) (string, error)
}
