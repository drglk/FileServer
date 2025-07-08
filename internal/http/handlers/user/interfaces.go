package user

import "context"

const pkg = "userHandler/"

type UserAdder interface {
	Register(ctx context.Context, login string, password string, token string) (string, error)
}
