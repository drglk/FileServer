package userservice

import (
	"context"
	"errors"
	"fileserver/internal/models"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUserAdder struct {
	mock.Mock
}

func (m *MockUserAdder) AddUser(ctx context.Context, user models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

type MockUserProvider struct {
	mock.Mock
}

func (m *MockUserProvider) UserByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserProvider) UserByLogin(ctx context.Context, login string) (*models.User, error) {
	args := m.Called(ctx, login)
	return args.Get(0).(*models.User), args.Error(1)
}

func TestAddUser_UniqueConstraint(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	user := models.User{Login: "test"}

	mockAdder.On("AddUser", mock.Anything, user).Return(&models.UniqueConstraintError{Constraint: "users_login_key"})

	err := service.AddUser(context.Background(), user)
	assert.ErrorIs(t, err, models.ErrUserExists)
}

func TestAddUser_OtherError(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	user := models.User{Login: "test"}

	mockAdder.On("AddUser", mock.Anything, user).Return(errors.New("db down"))

	err := service.AddUser(context.Background(), user)
	assert.ErrorIs(t, err, models.ErrFailedToAddUser)
}

func TestAddUser_Success(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	user := models.User{Login: "test"}

	mockAdder.On("AddUser", mock.Anything, user).Return(nil)

	err := service.AddUser(context.Background(), user)
	assert.NoError(t, err)
}

func TestUserByID_OtherError(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockProvider.On("UserByID", mock.Anything, "testID").Return((*models.User)(nil), errors.New("other error"))

	user, err := service.UserByID(context.Background(), "testID")
	assert.Nil(t, user)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestUserByID_NotFound(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockProvider.On("UserByID", mock.Anything, "testID").Return((*models.User)(nil), models.ErrUserNotFound)

	user, err := service.UserByID(context.Background(), "testID")
	assert.Nil(t, user)
	assert.ErrorIs(t, err, models.ErrUserNotFound)
}

func TestUserByID_Success(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockUser := models.User{ID: "testID", Login: "ghost", PassHash: []byte("passhash")}

	mockProvider.On("UserByID", mock.Anything, "testID").Return(&mockUser, nil)

	user, err := service.UserByID(context.Background(), "testID")

	assert.NoError(t, err)
	assert.Equal(t, mockUser.ID, user.ID)
	assert.Equal(t, mockUser.Login, user.Login)
	assert.Equal(t, mockUser.PassHash, user.PassHash)

	mockProvider.AssertExpectations(t)
}

func TestUserByLogin_OtherError(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockProvider.On("UserByLogin", mock.Anything, "ghost").Return((*models.User)(nil), errors.New("other error"))

	user, err := service.UserByLogin(context.Background(), "ghost")
	assert.Nil(t, user)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestUserByLogin_NotFound(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockProvider.On("UserByLogin", mock.Anything, "ghost").Return((*models.User)(nil), models.ErrUserNotFound)

	user, err := service.UserByLogin(context.Background(), "ghost")
	assert.Nil(t, user)
	assert.ErrorIs(t, err, models.ErrUserNotFound)
}

func TestUserByLogin_Success(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockUser := models.User{ID: "testID", Login: "ghost", PassHash: []byte("passhash")}

	mockProvider.On("UserByLogin", mock.Anything, "ghost").Return(&mockUser, nil)

	user, err := service.UserByLogin(context.Background(), "ghost")

	assert.NoError(t, err)
	assert.Equal(t, mockUser.ID, user.ID)
	assert.Equal(t, mockUser.Login, user.Login)
	assert.Equal(t, mockUser.PassHash, user.PassHash)

	mockProvider.AssertExpectations(t)
}

func TestUserIDByLogin_NotFound(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockProvider.On("UserByLogin", mock.Anything, "ghost").Return((*models.User)(nil), models.ErrUserNotFound)

	userID, err := service.UserIDByLogin(context.Background(), "ghost")
	assert.Equal(t, userID, "")
	assert.ErrorIs(t, err, models.ErrUserNotFound)
}

func TestUserIDByLogin_OtherError(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockProvider.On("UserByLogin", mock.Anything, "ghost").Return((*models.User)(nil), errors.New("other error"))

	userID, err := service.UserIDByLogin(context.Background(), "ghost")
	assert.Equal(t, userID, "")
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestUserIDByLogin_Success(t *testing.T) {
	t.Parallel()

	mockAdder := new(MockUserAdder)
	mockProvider := new(MockUserProvider)
	service := New(slog.Default(), mockAdder, mockProvider)

	mockUser := models.User{ID: "testID", Login: "ghost", PassHash: []byte("passhash")}

	mockProvider.On("UserByLogin", mock.Anything, "ghost").Return(&mockUser, nil)

	userID, err := service.UserIDByLogin(context.Background(), "ghost")

	assert.NoError(t, err)
	assert.Equal(t, mockUser.ID, userID)

	mockProvider.AssertExpectations(t)
}
