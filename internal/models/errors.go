package models

import (
	"errors"
	"fmt"
)

var (
	ErrNoRows                 = errors.New("no rows")
	ErrUNIQUEConstraintFailed = errors.New("unique constraint failed")
	ErrFailedToAddUser        = errors.New("failed to add user")
	ErrFailedToGetUser        = errors.New("failed to get user")
	ErrInternal               = errors.New("internal server error")
	ErrMethodNotAllowed       = errors.New("method not allowed")
	ErrForbidden              = errors.New("access denied")
	ErrInvalidParams          = errors.New("invalid params")
	ErrUserNotFound           = errors.New("user not found")
	ErrUserExists             = errors.New("user already exists")
	ErrDocumentNotFound       = errors.New("document not found")
	ErrSessionNotFound        = errors.New("sessions not found")
	ErrInvalidCredentials     = errors.New("invalid credentials")
)

type UniqueConstraintError struct {
	Constraint string
	Err        error
}

func (e *UniqueConstraintError) Error() string {
	return fmt.Sprintf("%v: %s", e.Err, e.Constraint)
}

func (e *UniqueConstraintError) Unwrap() error {
	return e.Err
}
