package session

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/models"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockSessionDeleter struct {
	mock.Mock
}

func (m *mockSessionDeleter) Logout(ctx context.Context, token string) error {
	args := m.Called(ctx, token)
	return args.Error(0)
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockDeleter := new(mockSessionDeleter)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	token := "token123"

	mockDeleter.On("Logout", ctx, token).Return(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/"+token, nil)

	Delete(ctx, logger, w, req, token, mockDeleter)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]map[string]bool
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["response"][token])

	mockDeleter.AssertExpectations(t)
}

func TestDelete_SessionNotFoundIgnored(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockDeleter := new(mockSessionDeleter)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	token := "missing"

	mockDeleter.On("Logout", ctx, token).Return(models.ErrSessionNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/"+token, nil)

	Delete(ctx, logger, w, req, token, mockDeleter)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]map[string]bool
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["response"][token])

	mockDeleter.AssertExpectations(t)
}

func TestDelete_UnexpectedError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockDeleter := new(mockSessionDeleter)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	token := "badtoken"

	mockDeleter.On("Logout", ctx, token).Return(errors.New("db failure"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/auth/"+token, nil)

	Delete(ctx, logger, w, req, token, mockDeleter)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]map[string]bool
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["response"][token])

	mockDeleter.AssertExpectations(t)
}
