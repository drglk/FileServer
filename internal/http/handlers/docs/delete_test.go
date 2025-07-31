package docs

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/models"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockDocDeleter struct{ mock.Mock }

func (m *mockDocDeleter) DeleteDocument(ctx context.Context, docID string, user *models.User) error {
	args := m.Called(ctx, docID, user)
	return args.Error(0)
}

func TestDelete_Success(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/docs/doc42?token=tok123", nil)

	user := &models.User{ID: "u1"}
	docID := "doc42"
	token := "tok123"

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	deleter := new(mockDocDeleter)
	deleter.On("DeleteDocument", ctx, docID, user).Return(nil)

	Delete(ctx, slog.Default(), w, req, docID, deleter, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var parsed map[string]map[string]bool
	err := json.NewDecoder(resp.Body).Decode(&parsed)
	assert.NoError(t, err)
	assert.Equal(t, true, parsed["response"][token])

	deleter.AssertExpectations(t)
}

func TestDelete_Fail_InvalidToken(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/docs/doc42?token=bad", nil)
	ctx := req.Context()

	Delete(ctx, slog.Default(), w, req, "doc42", nil, nil)
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
}

func TestDelete_Fail_Forbidden(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/docs/doc42?token=tok", nil)

	user := &models.User{ID: "u2"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	deleter := new(mockDocDeleter)
	deleter.On("DeleteDocument", ctx, "doc42", user).Return(models.ErrForbidden)

	Delete(ctx, slog.Default(), w, req, "doc42", deleter, nil)
	assert.Equal(t, http.StatusForbidden, w.Result().StatusCode)
	deleter.AssertExpectations(t)
}

func TestDelete_Fail_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/docs/doc42?token=tok", nil)

	user := &models.User{ID: "u2"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	deleter := new(mockDocDeleter)
	deleter.On("DeleteDocument", ctx, "doc42", user).Return(errors.New("unexpected"))

	Delete(ctx, slog.Default(), w, req, "doc42", deleter, nil)
	assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	deleter.AssertExpectations(t)
}
