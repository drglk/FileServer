package authservice

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/models"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type MockUserAdder struct {
	mock.Mock
}

func (m *MockUserAdder) AddUser(ctx context.Context, user models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

// MockUserProvider mocks the UserProvider interface
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

// MockSessionStorer mocks the SessionStorer interface
type MockSessionStorer struct {
	mock.Mock
}

func (m *MockSessionStorer) SaveSession(ctx context.Context, token string, userJSON string) error {
	args := m.Called(ctx, token, userJSON)
	return args.Error(0)
}

func (m *MockSessionStorer) DeleteSession(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func (m *MockSessionStorer) UserByToken(ctx context.Context, token string) (string, error) {
	args := m.Called(ctx, token)
	return args.String(0), args.Error(1)
}

func TestRegister_InvalidAdminToken(t *testing.T) {
	service := New(
		slog.Default(),
		new(MockUserAdder),
		nil,
		nil,
		"correct-token",
	)

	login, err := service.Register(context.Background(), "user", "pass1234", "wrong-token")
	assert.ErrorIs(t, err, models.ErrForbidden)
	assert.Empty(t, login)
}

func TestRegister_InvalidCredentials(t *testing.T) {
	service := New(
		slog.Default(),
		new(MockUserAdder),
		nil,
		nil,
		"admin-token",
	)

	login, err := service.Register(context.Background(), "", "", "admin-token")
	assert.ErrorIs(t, err, models.ErrInvalidParams)
	assert.Empty(t, login)
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	mockAdder := new(MockUserAdder)
	service := New(
		slog.Default(),
		mockAdder,
		nil,
		nil,
		"admin-token",
	)

	mockAdder.On("AddUser", mock.Anything, mock.AnythingOfType("models.User")).
		Return(models.ErrUserExists)

	login, err := service.Register(context.Background(), "useruser1", "validPass123!", "admin-token")
	assert.ErrorIs(t, err, models.ErrUserExists)
	assert.Empty(t, login)
}

func TestRegister_UnexpectedAddError(t *testing.T) {
	mockAdder := new(MockUserAdder)
	service := New(
		slog.Default(),
		mockAdder,
		nil,
		nil,
		"admin-token",
	)

	mockAdder.On("AddUser", mock.Anything, mock.AnythingOfType("models.User")).
		Return(errors.New("db down"))

	login, err := service.Register(context.Background(), "useruser1", "validPass123!", "admin-token")
	assert.ErrorIs(t, err, models.ErrInternal)
	assert.Empty(t, login)
}

func TestRegister_Success(t *testing.T) {
	mockAdder := new(MockUserAdder)
	service := New(
		slog.Default(),
		mockAdder,
		nil,
		nil,
		"admin-token",
	)

	mockAdder.On("AddUser", mock.Anything, mock.MatchedBy(func(u models.User) bool {
		err := bcrypt.CompareHashAndPassword(u.PassHash, []byte("validPass123!"))
		return u.Login == "useruser1" && err == nil
	})).Return(nil)

	login, err := service.Register(context.Background(), "useruser1", "validPass123!", "admin-token")
	assert.NoError(t, err)
	assert.Equal(t, "useruser1", login)
}

func hash(t *testing.T, password string) []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	return hash
}

func TestLogin_UserNotFound(t *testing.T) {
	mockUsers := new(MockUserProvider)
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, mockUsers, mockSessions, "")

	mockUsers.On("UserByLogin", mock.Anything, "ghost").
		Return((*models.User)(nil), models.ErrUserNotFound)

	token, err := service.Login(context.Background(), "ghost", "pass")
	assert.ErrorIs(t, err, models.ErrUserNotFound)
	assert.Empty(t, token)
}

func TestLogin_UserProviderFails(t *testing.T) {
	mockUsers := new(MockUserProvider)
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, mockUsers, mockSessions, "")

	mockUsers.On("UserByLogin", mock.Anything, "ghost").
		Return((*models.User)(nil), errors.New("db fail"))

	token, err := service.Login(context.Background(), "ghost", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Login")
	assert.Empty(t, token)
}

func TestLogin_InvalidPassword(t *testing.T) {
	mockUsers := new(MockUserProvider)
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, mockUsers, mockSessions, "")

	mockUsers.On("UserByLogin", mock.Anything, "ghost").
		Return(&models.User{
			ID:       "u1",
			Login:    "ghost",
			PassHash: hash(t, "correctpass"),
		}, nil)

	token, err := service.Login(context.Background(), "ghost", "wrongpass")
	assert.ErrorIs(t, err, models.ErrInvalidCredentials)
	assert.Empty(t, token)
}

func TestLogin_SessionStoreFails(t *testing.T) {
	mockUsers := new(MockUserProvider)
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, mockUsers, mockSessions, "")

	user := &models.User{
		ID:       "u1",
		Login:    "ghost",
		PassHash: hash(t, "pass"),
	}

	mockUsers.On("UserByLogin", mock.Anything, "ghost").
		Return(user, nil)

	marshaled, _ := json.Marshal(user)
	mockSessions.On("SaveSession", mock.Anything, mock.AnythingOfType("string"), string(marshaled)).
		Return(errors.New("redis fail"))

	token, err := service.Login(context.Background(), "ghost", "pass")
	assert.ErrorIs(t, err, models.ErrInternal)
	assert.Empty(t, token)
}

func TestLogin_Success(t *testing.T) {
	mockUsers := new(MockUserProvider)
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, mockUsers, mockSessions, "")

	user := &models.User{
		ID:       "u1",
		Login:    "ghost",
		PassHash: hash(t, "pass"),
	}

	mockUsers.On("UserByLogin", mock.Anything, "ghost").
		Return(user, nil)

	mockSessions.On("SaveSession", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(nil)

	token, err := service.Login(context.Background(), "ghost", "pass")
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestUserByToken_SessionNotFound(t *testing.T) {
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, nil, mockSessions, "")

	mockSessions.On("UserByToken", mock.Anything, "badtoken").
		Return("", models.ErrSessionNotFound)

	user, err := service.UserByToken(context.Background(), "badtoken")
	assert.ErrorIs(t, err, models.ErrInvalidCredentials)
	assert.Nil(t, user)
}

func TestUserByToken_SessionFails(t *testing.T) {
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, nil, mockSessions, "")

	mockSessions.On("UserByToken", mock.Anything, "token123").
		Return("", errors.New("redis down"))

	user, err := service.UserByToken(context.Background(), "token123")
	assert.ErrorIs(t, err, models.ErrInternal)
	assert.Nil(t, user)
}

func TestUserByToken_UnmarshalFails(t *testing.T) {
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, nil, mockSessions, "")

	mockSessions.On("UserByToken", mock.Anything, "token123").
		Return("invalid-json", nil)

	user, err := service.UserByToken(context.Background(), "token123")
	assert.ErrorIs(t, err, models.ErrInternal)
	assert.Nil(t, user)
}

func TestUserByToken_Success(t *testing.T) {
	mockSessions := new(MockSessionStorer)

	service := New(slog.Default(), nil, nil, mockSessions, "")

	user := models.User{
		ID:    "u1",
		Login: "ghost",
	}
	userJSON, _ := json.Marshal(user)

	mockSessions.On("UserByToken", mock.Anything, "token123").
		Return(string(userJSON), nil)

	res, err := service.UserByToken(context.Background(), "token123")
	assert.NoError(t, err)
	assert.Equal(t, user.ID, res.ID)
	assert.Equal(t, user.Login, res.Login)
}
func TestLogout_SessionNotFound(t *testing.T) {
	mockSessions := new(MockSessionStorer)
	service := New(slog.Default(), nil, nil, mockSessions, "")

	mockSessions.On("DeleteSession", mock.Anything, "badtoken").
		Return(models.ErrSessionNotFound)

	err := service.Logout(context.Background(), "badtoken")
	assert.ErrorIs(t, err, models.ErrSessionNotFound)
}

func TestLogout_StorageError(t *testing.T) {
	mockSessions := new(MockSessionStorer)
	service := New(slog.Default(), nil, nil, mockSessions, "")

	mockSessions.On("DeleteSession", mock.Anything, "token123").
		Return(errors.New("redis error"))

	err := service.Logout(context.Background(), "token123")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "Logout")
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestLogout_Success(t *testing.T) {
	mockSessions := new(MockSessionStorer)
	service := New(slog.Default(), nil, nil, mockSessions, "")

	mockSessions.On("DeleteSession", mock.Anything, "token123").
		Return(nil)

	err := service.Logout(context.Background(), "token123")
	assert.NoError(t, err)
}
