package user

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/models"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUserAdder struct {
	mock.Mock
}

func (m *mockUserAdder) Register(ctx context.Context, login string, password string, token string) (string, error) {
	args := m.Called(ctx, login, password, token)
	return args.String(0), args.Error(1)
}

func TestAdd_Success(t *testing.T) {
	t.Parallel()

	body := `{"login": "user1", "pswd": "pass123", "token": "admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockUserAdder)
	mockAdder.On("Register", mock.Anything, "user1", "pass123", "admin").Return("user1", nil)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Add(req.Context(), logger, w, req, mockAdder)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed map[string]map[string]string
	err := json.NewDecoder(resp.Body).Decode(&parsed)
	assert.NoError(t, err)
	assert.Equal(t, "user1", parsed["response"]["login"])
	mockAdder.AssertExpectations(t)
}

func TestAdd_InvalidJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(`{invalid json}`))
	w := httptest.NewRecorder()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	Add(req.Context(), logger, w, req, new(mockUserAdder))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAdd_UserExists(t *testing.T) {
	t.Parallel()

	body := `{"login": "existing", "pswd": "pass", "token": "token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockUserAdder)
	mockAdder.On("Register", mock.Anything, "existing", "pass", "token").Return("", models.ErrUserExists)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	Add(req.Context(), logger, w, req, mockAdder)

	assert.Equal(t, http.StatusConflict, w.Code)
	mockAdder.AssertExpectations(t)
}

func TestAdd_Forbidden(t *testing.T) {
	t.Parallel()

	body := `{"login": "any", "pswd": "pass", "token": "bad"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockUserAdder)
	mockAdder.On("Register", mock.Anything, "any", "pass", "bad").Return("", models.ErrForbidden)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	Add(req.Context(), logger, w, req, mockAdder)

	assert.Equal(t, http.StatusForbidden, w.Code)
	mockAdder.AssertExpectations(t)
}

func TestAdd_InternalError(t *testing.T) {
	t.Parallel()

	body := `{"login": "fail", "pswd": "pass", "token": "token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockUserAdder)
	mockAdder.On("Register", mock.Anything, "fail", "pass", "token").Return("", errors.New("db down"))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	Add(req.Context(), logger, w, req, mockAdder)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockAdder.AssertExpectations(t)
}

func TestAdd_InvalidParams(t *testing.T) {
	t.Parallel()

	body := `{"login": "user1", "pswd": "", "token": "admin"}`
	req := httptest.NewRequest(http.MethodPost, "/api/register", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockUserAdder)
	mockAdder.On("Register", mock.Anything, "user1", "", "admin").
		Return("", models.ErrInvalidParams)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Add(req.Context(), logger, w, req, mockAdder)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	mockAdder.AssertExpectations(t)
}
