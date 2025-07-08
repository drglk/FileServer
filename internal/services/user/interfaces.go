package userservice

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
