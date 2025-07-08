package session

import (
	"context"
)

const pkg = "sessionHandler/"

type SessionAdder interface {
	Login(ctx context.Context, login string, password string) (string, error)
}

type SessionDeleter interface {
	Logout(ctx context.Context, token string) error
}
