package docs

import (
	"context"
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

type mockDocProvider struct{ mock.Mock }

func (m *mockDocProvider) ListDocuments(ctx context.Context, requester *models.User, login string, filter models.DocumentFilter) ([]*models.Document, error) {
	args := m.Called(ctx, requester, login, filter)
	return args.Get(0).([]*models.Document), args.Error(1)
}

func (m *mockDocProvider) DocumentByID(ctx context.Context, docID string, requester *models.User) (*models.Document, io.ReadCloser, error) {
	args := m.Called(ctx, docID, requester)
	doc := args.Get(0)
	var reader io.ReadCloser
	if rc, ok := args.Get(1).(io.ReadCloser); ok {
		reader = rc
	}
	return doc.(*models.Document), reader, args.Error(2)
}

func TestHead_Success(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodHead, "/api/docs?token=valid&key=name&value=test&limit=2", nil)
	w := httptest.NewRecorder()

	user := &models.User{ID: "user-id"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	doc := &models.Document{Name: "test"}
	docProvider := new(mockDocProvider)
	docProvider.On("ListDocuments", mock.Anything, user, "", models.DocumentFilter{
		Key: "name", Value: "test", Limit: 2,
	}).Return([]*models.Document{doc}, nil)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	Head(ctx, logger, w, req, docProvider, new(mockUserIDProvider))

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "1", resp.Header.Get("X-Documents-Count"))

	docProvider.AssertExpectations(t)
}

func TestHead_Forbidden_InvalidToken(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodHead, "/api/docs?token=bad", nil)
	w := httptest.NewRecorder()
	ctx := req.Context()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	Head(ctx, logger, w, req, new(mockDocProvider), new(mockUserIDProvider))

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestHead_InternalError_ListFail(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodHead, "/api/docs?token=valid", nil)
	w := httptest.NewRecorder()

	user := &models.User{ID: "user-id"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	docProvider := new(mockDocProvider)
	docProvider.On("ListDocuments", mock.Anything, user, "", models.DocumentFilter{}).
		Return([]*models.Document{}, errors.New("db error"))

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	Head(ctx, logger, w, req, docProvider, new(mockUserIDProvider))

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestHeadByID_Success_File(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/docs/doc123?token=token123", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	doc := &models.Document{
		ID:       "doc123",
		Name:     "report.pdf",
		Mime:     "application/pdf",
		IsFile:   true,
		IsPublic: false,
		OwnerID:  "user1",
	}

	dp := new(mockDocProvider)
	dp.On("DocumentByID", ctx, "doc123", user).Return(doc, io.NopCloser(strings.NewReader("")), nil)

	HeadByID(ctx, slog.Default(), w, req, "doc123", dp, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	assert.Equal(t, `attachment; filename="report.pdf"`, resp.Header.Get("Content-Disposition"))
	assert.Equal(t, "application/pdf", resp.Header.Get("X-Content-Mime"))

	dp.AssertExpectations(t)
}

func TestHeadByID_Success_JSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/docs/doc456?token=token456", nil)

	user := &models.User{ID: "user2"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	doc := &models.Document{
		ID:       "doc456",
		Name:     "metadata",
		Mime:     "application/json",
		IsFile:   false,
		IsPublic: true,
		OwnerID:  "user2",
	}

	dp := new(mockDocProvider)
	dp.On("DocumentByID", ctx, "doc456", user).Return(doc, io.NopCloser(strings.NewReader("")), nil)

	HeadByID(ctx, slog.Default(), w, req, "doc456", dp, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "application/json", resp.Header.Get("X-Content-Mime"))

	dp.AssertExpectations(t)
}

func TestHeadByID_Fail_InvalidToken(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/docs/doc123?token=badtoken", nil)
	ctx := req.Context()

	dp := new(mockDocProvider)

	HeadByID(ctx, slog.Default(), w, req, "doc123", dp, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestHeadByID_Fail_Forbidden(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/docs/doc123?token=token123", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	dp := new(mockDocProvider)

	mockFile := new(io.ReadCloser)

	dp.On("DocumentByID", ctx, "doc123", user).Return((*models.Document)(nil), mockFile, models.ErrForbidden)

	HeadByID(ctx, slog.Default(), w, req, "doc123", dp, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	dp.AssertExpectations(t)
}

func TestHeadByID_Fail_UnknownError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/docs/doc123?token=token123", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	dp := new(mockDocProvider)

	mockFile := new(io.ReadCloser)

	dp.On("DocumentByID", ctx, "doc123", user).Return((*models.Document)(nil), mockFile, errors.New("db error"))

	HeadByID(ctx, slog.Default(), w, req, "doc123", dp, nil)

	resp := w.Result()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	dp.AssertExpectations(t)
}
