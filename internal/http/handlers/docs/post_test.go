package docs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/dto"
	"fileserver/internal/models"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAuth struct {
	mock.Mock
}

func (m *mockAuth) UserByToken(ctx context.Context, token string) (*models.User, error) {
	args := m.Called(ctx, token)
	return args.Get(0).(*models.User), args.Error(1)
}

type mockUploader struct {
	mock.Mock
}

func (m *mockUploader) UploadDocument(ctx context.Context, user *models.User, doc *models.Document, content io.Reader) (string, error) {
	args := m.Called(ctx, user, doc, mock.Anything)
	return args.String(0), args.Error(1)
}

type mockUserIDProvider struct {
	mock.Mock
}

func (m *mockUserIDProvider) UserIDByLogin(ctx context.Context, login string) (string, error) {
	args := m.Called(ctx, login)
	return args.String(0), args.Error(1)
}

func TestUpload_Success(t *testing.T) {
	t.Parallel()

	bodyBuf := &bytes.Buffer{}
	writer := multipart.NewWriter(bodyBuf)

	metaBytes, err := os.ReadFile("test_data/meta.json")
	assert.NoError(t, err)
	assert.NoError(t, writer.WriteField("meta", string(metaBytes)))

	jsonFile, err := os.Open("test_data/json.json")
	assert.NoError(t, err)
	jsonPart, err := writer.CreateFormFile("json", "json.json")
	assert.NoError(t, err)
	_, err = io.Copy(jsonPart, jsonFile)
	assert.NoError(t, err)
	jsonFile.Close()

	file, err := os.Open("test_data/file.txt")
	assert.NoError(t, err)
	filePart, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	assert.NoError(t, err)
	_, err = io.Copy(filePart, file)
	assert.NoError(t, err)
	file.Close()

	assert.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/docs", bodyBuf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	auth := new(mockAuth)
	uploader := new(mockUploader)
	userID := new(mockUserIDProvider)

	user := &models.User{ID: "user1", Login: "user1"}
	auth.On("UserByToken", mock.Anything, "valid-token").Return(user, nil)
	userID.On("UserIDByLogin", mock.Anything, "user2").Return("user2-id", nil)

	uploader.On("UploadDocument", mock.Anything, user, mock.AnythingOfType("*models.Document"), mock.Anything).Return("doc-id", nil)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Upload(req.Context(), logger, w, req, auth, uploader, userID)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var parsed map[string]map[string]json.RawMessage
	err = json.NewDecoder(resp.Body).Decode(&parsed)
	assert.NoError(t, err)
	assert.Contains(t, parsed["data"], "json")
	assert.Contains(t, parsed["data"], "file")

	auth.AssertExpectations(t)
	uploader.AssertExpectations(t)
	userID.AssertExpectations(t)
}

func TestUpload_ParseMultipartFormError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/api/docs", strings.NewReader("invalid"))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=----badboundary")
	w := httptest.NewRecorder()

	auth := new(mockAuth)
	du := new(mockUploader)
	up := new(mockUserIDProvider)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Upload(req.Context(), logger, w, req, auth, du, up)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpload_InvalidMetaJSON(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	writer.WriteField("meta", "invalid-json")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/docs", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	auth := new(mockAuth)
	du := new(mockUploader)
	up := new(mockUserIDProvider)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Upload(req.Context(), logger, w, req, auth, du, up)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpload_InvalidToken(t *testing.T) {
	t.Parallel()

	meta := dto.UploadMeta{
		Token: "bad-token",
	}
	metaJSON, _ := json.Marshal(meta)

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	writer.WriteField("meta", string(metaJSON))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/docs", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	auth := new(mockAuth)
	auth.On("UserByToken", mock.Anything, "bad-token").Return((*models.User)(nil), errors.New("unauthorized"))

	du := new(mockUploader)
	up := new(mockUserIDProvider)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Upload(req.Context(), logger, w, req, auth, du, up)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpload_InvalidJSONFile(t *testing.T) {
	t.Parallel()

	meta := dto.UploadMeta{
		Token:    "valid",
		Name:     "test.json",
		Mime:     "application/json",
		IsFile:   false,
		IsPublic: false,
		Grants:   nil,
	}
	metaJSON, _ := json.Marshal(meta)

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	writer.WriteField("meta", string(metaJSON))

	jsonPart, _ := writer.CreateFormFile("json", "data.json")
	jsonPart.Write([]byte("{invalid json"))

	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/docs", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	auth := new(mockAuth)
	auth.On("UserByToken", mock.Anything, "valid").Return(&models.User{ID: "user1"}, nil)

	du := new(mockUploader)
	up := new(mockUserIDProvider)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Upload(req.Context(), logger, w, req, auth, du, up)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpload_FileRequiredButMissing(t *testing.T) {
	t.Parallel()

	meta := dto.UploadMeta{
		Token:    "valid",
		Name:     "test.txt",
		Mime:     "text/plain",
		IsFile:   true,
		IsPublic: true,
	}
	metaJSON, _ := json.Marshal(meta)

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	writer.WriteField("meta", string(metaJSON))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/docs", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	auth := new(mockAuth)
	auth.On("UserByToken", mock.Anything, "valid").Return(&models.User{ID: "user1"}, nil)

	du := new(mockUploader)
	up := new(mockUserIDProvider)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Upload(req.Context(), logger, w, req, auth, du, up)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpload_UploadDocumentFails(t *testing.T) {
	meta := dto.UploadMeta{
		Token:    "valid",
		Name:     "doc.txt",
		Mime:     "text/plain",
		IsFile:   false,
		IsPublic: false,
		Grants:   nil,
	}
	metaJSON, _ := json.Marshal(meta)

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	writer.WriteField("meta", string(metaJSON))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/docs", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	auth := new(mockAuth)
	auth.On("UserByToken", mock.Anything, "valid").Return(&models.User{ID: "user1"}, nil)

	du := new(mockUploader)

	du.On("UploadDocument", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("upload failed"))

	up := new(mockUserIDProvider)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	Upload(req.Context(), logger, w, req, auth, du, up)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}
