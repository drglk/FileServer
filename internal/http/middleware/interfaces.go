package middleware

import (
	"context"
	"fileserver/internal/models"
)

const pkg = "middleware/"

type SessionStorer interface {
	UserByToken(ctx context.Context, token string) (*models.User, error)
}
