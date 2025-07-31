package docs

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/dto"
	"fileserver/internal/models"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGet_Success(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs?token=valid-token", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	documents := []*models.Document{
		{
			ID:        "doc1",
			Name:      "file1.txt",
			Mime:      "text/plain",
			IsFile:    true,
			IsPublic:  true,
			CreatedAt: time.Now(),
			Grants:    []string{"user2"},
		},
	}

	mockDP := new(mockDocProvider)
	mockDP.On("ListDocuments", ctx, user, "", mock.AnythingOfType("models.DocumentFilter")).Return(documents, nil)

	Get(ctx, slog.Default(), w, req, mockDP, nil)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var parsed map[string]map[string][]dto.DocumentResponse
	err := json.NewDecoder(resp.Body).Decode(&parsed)
	assert.NoError(t, err)
	assert.Len(t, parsed["data"]["docs"], 1)
	assert.Equal(t, "doc1", parsed["data"]["docs"][0].ID)

	mockDP.AssertExpectations(t)
}

func TestGet_Fail_Unauthorized(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs?token=bad-token", nil)
	ctx := req.Context()

	Get(ctx, slog.Default(), w, req, nil, nil)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestGet_Fail_ListDocumentsError(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs?token=valid-token", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	mockDP := new(mockDocProvider)
	mockDP.On("ListDocuments", ctx, user, "", mock.AnythingOfType("models.DocumentFilter")).Return([]*models.Document{}, errors.New("db error"))

	Get(ctx, slog.Default(), w, req, mockDP, nil)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	mockDP.AssertExpectations(t)
}

func TestGetByID_Success_File(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs/doc123?token=valid-token", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	doc := &models.Document{
		ID:     "doc123",
		Name:   "example.pdf",
		Mime:   "application/pdf",
		IsFile: true,
	}

	fileContent := "file data"
	fileReader := io.NopCloser(strings.NewReader(fileContent))

	dp := new(mockDocProvider)
	dp.On("DocumentByID", ctx, "doc123", user).Return(doc, fileReader, nil)

	GetByID(ctx, slog.Default(), w, req, "doc123", dp, nil)
	resp := w.Result()
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/pdf", resp.Header.Get("Content-Type"))
	assert.Equal(t, "attachment; filename=\"example.pdf\"", resp.Header.Get("Content-Disposition"))
	assert.Equal(t, fileContent, string(data))

	dp.AssertExpectations(t)
}

func TestGetByID_Success_JSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs/doc456?token=valid-token", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	jsonBody := []byte(`{"key": "value"}`)
	doc := &models.Document{
		ID:       "doc456",
		Name:     "meta.json",
		Mime:     "application/json",
		IsFile:   false,
		JSONData: jsonBody,
	}

	dp := new(mockDocProvider)
	dp.On("DocumentByID", ctx, "doc456", user).Return(doc, nil, nil)

	GetByID(ctx, slog.Default(), w, req, "doc456", dp, nil)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var result map[string]map[string]string
	err := json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "value", result["data"]["key"])

	dp.AssertExpectations(t)
}

func TestGetByID_Fail_Forbidden(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs/doc123?token=invalid-token", nil)
	ctx := req.Context()

	GetByID(ctx, slog.Default(), w, req, "doc123", nil, nil)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestGetByID_Fail_BadParams(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs/doc123?token=valid-token", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	dp := new(mockDocProvider)
	dp.On("DocumentByID", ctx, "doc123", user).Return((*models.Document)(nil), nil, errors.New("bad param"))

	GetByID(ctx, slog.Default(), w, req, "doc123", dp, nil)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	dp.AssertExpectations(t)
}

func TestGetByID_Fail_InvalidJSON(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/docs/doc789?token=valid-token", nil)

	user := &models.User{ID: "user1"}

	ctx := context.WithValue(req.Context(), models.UserContextKey, user)

	req = req.WithContext(ctx)

	doc := &models.Document{
		ID:       "doc789",
		Name:     "corrupt.json",
		Mime:     "application/json",
		IsFile:   false,
		JSONData: []byte(`{invalid json}`),
	}

	dp := new(mockDocProvider)
	dp.On("DocumentByID", ctx, "doc789", user).Return(doc, nil, nil)

	GetByID(ctx, slog.Default(), w, req, "doc789", dp, nil)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	dp.AssertExpectations(t)
}
