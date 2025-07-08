package userservice

import (
	"context"
	"errors"
	"fileserver/internal/models"
	"log/slog"
)

const pkg = "userService/"

type UserService struct {
	log          *slog.Logger
	userAdder    UserAdder
	userProvider UserProvider
}

func New(
	log *slog.Logger,
	userAdder UserAdder,
	userProvider UserProvider) *UserService {
	return &UserService{
		log:          log,
		userAdder:    userAdder,
		userProvider: userProvider,
	}
}

func (u *UserService) AddUser(ctx context.Context, user models.User) error {
	op := pkg + "AddUser"

	log := u.log.With(slog.String("op", op))

	log.Debug("attempting to add user")

	err := u.userAdder.AddUser(ctx, user)
	if err != nil {
		var uce *models.UniqueConstraintError
		if errors.As(err, &uce) {
			log.Warn("user already exists", slog.String("constraint", uce.Constraint))
			return models.ErrUserExists
		}
		log.Error("failed to add user", slog.String("error", err.Error()))
		return models.ErrFailedToAddUser
	}

	log.Debug("user added successfully")

	return nil
}

func (u *UserService) UserByID(ctx context.Context, id string) (*models.User, error) {
	op := pkg + "UserByID"

	log := u.log.With(slog.String("op", op))

	log.Debug("attempting to get user by id")

	user, err := u.userProvider.UserByID(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			log.Warn("failed to get user by id", slog.String("error", models.ErrUserNotFound.Error()))
			return nil, models.ErrUserNotFound
		}
		log.Error("failed to get user by id", slog.String("error", err.Error()))
		return nil, models.ErrInternal
	}

	log.Debug("user founded successfully")

	return user, nil
}

func (u *UserService) UserByLogin(ctx context.Context, login string) (*models.User, error) {
	op := pkg + "UserByLogin"

	log := u.log.With(slog.String("op", op))

	log.Debug("attempting to get user by login")

	user, err := u.userProvider.UserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			log.Warn("failed to get user by login", slog.String("error", models.ErrUserNotFound.Error()))
			return nil, models.ErrUserNotFound
		}
		log.Error("failed to get user by login", slog.String("error", err.Error()))
		return nil, models.ErrInternal
	}

	log.Debug("user founded successfully")

	return user, nil
}

func (u *UserService) UserIDByLogin(ctx context.Context, login string) (string, error) {
	op := pkg + "UserIDByLogin"

	log := u.log.With(slog.String("op", op))

	log.Debug("attempting to get user id by login")

	user, err := u.userProvider.UserByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, models.ErrUserNotFound) {
			log.Warn("failed to get user by login", slog.String("error", models.ErrUserNotFound.Error()))
			return "", models.ErrUserNotFound
		}
		log.Error("failed to get user by login", slog.String("error", err.Error()))
		return "", models.ErrInternal
	}

	log.Debug("user founded successfully")

	return user.ID, nil
}
