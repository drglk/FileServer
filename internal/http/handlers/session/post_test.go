package session

import (
	"context"
	"encoding/json"
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

type mockSessionAdder struct {
	mock.Mock
}

func (m *mockSessionAdder) Login(ctx context.Context, login, password string) (string, error) {
	args := m.Called(ctx, login, password)
	return args.String(0), args.Error(1)
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestSessionAdd_Success(t *testing.T) {
	t.Parallel()

	body := `{"login": "user1", "pswd": "secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockSessionAdder)
	mockAdder.On("Login", mock.Anything, "user1", "secret").Return("mocked-token", nil)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Add(req.Context(), logger, w, req, mockAdder)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]map[string]string
	err := json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "mocked-token", result["response"]["token"])

	mockAdder.AssertExpectations(t)
}

func TestSessionAdd_ReadBodyError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/auth", errReader{})
	w := httptest.NewRecorder()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockAdder := new(mockSessionAdder)

	Add(req.Context(), logger, w, req, mockAdder)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestSessionAdd_InvalidJSON(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/auth", strings.NewReader("{invalid json"))
	w := httptest.NewRecorder()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	mockAdder := new(mockSessionAdder)

	Add(req.Context(), logger, w, req, mockAdder)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestSessionAdd_UserNotFound(t *testing.T) {
	t.Parallel()

	body := `{"login": "ghost", "pswd": "nopass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockSessionAdder)
	mockAdder.On("Login", mock.Anything, "ghost", "nopass").
		Return("", models.ErrUserNotFound)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Add(req.Context(), logger, w, req, mockAdder)

	resp := w.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSessionAdd_InvalidCredentials(t *testing.T) {
	t.Parallel()

	body := `{"login": "user1", "pswd": "wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth", strings.NewReader(body))
	w := httptest.NewRecorder()

	mockAdder := new(mockSessionAdder)
	mockAdder.On("Login", mock.Anything, "user1", "wrong").
		Return("", models.ErrInvalidCredentials)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Add(req.Context(), logger, w, req, mockAdder)

	resp := w.Result()
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
