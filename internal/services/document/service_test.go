package documentservice

import (
	"bytes"
	"context"
	"errors"
	"fileserver/internal/models"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDocumentRepository struct {
	mock.Mock
}

func (m *MockDocumentRepository) CreateDocument(ctx context.Context, doc *models.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockDocumentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockDocumentRepository) DocumentByID(ctx context.Context, id string) (*models.Document, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Document), args.Error(1)
}

func (m *MockDocumentRepository) DocumentsGrantedTo(ctx context.Context, userID string) ([]*models.Document, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.Document), args.Error(1)
}

func (m *MockDocumentRepository) GrantAccess(ctx context.Context, docID string, userIDs []string) error {
	args := m.Called(ctx, docID, userIDs)
	return args.Error(0)
}

func (m *MockDocumentRepository) ListByUser(ctx context.Context, userID string) ([]*models.Document, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*models.Document), args.Error(1)
}

func (m *MockDocumentRepository) FilteredDocuments(ctx context.Context, login string, requesterID string, filter models.DocumentFilter) ([]*models.Document, error) {
	args := m.Called(ctx, login, requesterID, filter)
	return args.Get(0).([]*models.Document), args.Error(1)
}

type MockFileStorage struct {
	mock.Mock
}

func (m *MockFileStorage) SaveFile(doc *models.Document, reader io.Reader) error {
	args := m.Called(doc, reader)
	return args.Error(0)
}

func (m *MockFileStorage) LoadFile(doc *models.Document) (io.ReadCloser, error) {
	args := m.Called(doc)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockFileStorage) DeleteFile(doc *models.Document) error {
	args := m.Called(doc)
	return args.Error(0)
}

type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockCache) Del(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func TestUploadDocument_Success_FileWithGrants(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		Name:    "test.txt",
		IsFile:  true,
		OwnerID: "user1",
		Grants:  []string{"u1", "u2"},
	}
	user := &models.User{Login: "owner"}

	mockStorage.On("SaveFile", doc, mock.Anything).Return(nil)
	mockRepo.On("CreateDocument", ctx, doc).Return(nil)
	mockRepo.On("GrantAccess", ctx, mock.Anything, doc.Grants).Return(nil)
	mockCache.On("Del", ctx, []string{"docs:u1"}).Return(nil)
	mockCache.On("Del", ctx, []string{"docs:u2"}).Return(nil)
	mockCache.On("Del", ctx, []string{"docs:owner"}).Return(nil)

	id, err := service.UploadDocument(ctx, user, doc, bytes.NewReader([]byte("data")))

	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	mockStorage.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestUploadDocument_SaveFileError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{Name: "fail.txt", IsFile: true}
	user := &models.User{Login: "x"}

	mockStorage.On("SaveFile", doc, mock.Anything).Return(errors.New("disk error"))

	id, err := service.UploadDocument(ctx, user, doc, bytes.NewReader([]byte("data")))
	assert.ErrorContains(t, err, "UploadDocument")
	assert.Empty(t, id)
}

func TestUploadDocument_CreateMetadataError_FileDeleted(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{Name: "meta_fail.txt", IsFile: true}
	user := &models.User{Login: "x"}

	mockStorage.On("SaveFile", doc, mock.Anything).Return(nil)
	mockRepo.On("CreateDocument", ctx, doc).Return(errors.New("db error"))
	mockStorage.On("DeleteFile", doc).Return(nil)

	id, err := service.UploadDocument(ctx, user, doc, bytes.NewReader([]byte("data")))
	assert.ErrorContains(t, err, "UploadDocument")
	assert.Empty(t, id)
}

func TestUploadDocument_GrantAccessError_Cleanup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		Name:    "grants_fail.txt",
		IsFile:  true,
		OwnerID: "owner",
		Grants:  []string{"x"},
	}
	user := &models.User{Login: "owner"}

	mockStorage.On("SaveFile", doc, mock.Anything).Return(nil)
	mockRepo.On("CreateDocument", ctx, doc).Return(nil)
	mockRepo.On("GrantAccess", ctx, mock.Anything, doc.Grants).Return(errors.New("grant fail"))
	mockRepo.On("Delete", ctx, mock.Anything).Return(nil)
	mockStorage.On("DeleteFile", doc).Return(nil)

	id, err := service.UploadDocument(ctx, user, doc, bytes.NewReader([]byte("data")))
	assert.ErrorContains(t, err, "UploadDocument")
	assert.Empty(t, id)
}

func TestDocumentMetaByID_FromCacheSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{ID: "doc1", Name: "cached"}
	jsonStr, _ := docToJSON(doc)

	mockCache.On("Get", ctx, "doc1").Return(jsonStr, nil)

	res, err := service.documentMetaByID(ctx, "doc1")

	assert.NoError(t, err)
	assert.Equal(t, doc.ID, res.ID)
	mockCache.AssertExpectations(t)
}

func TestDocumentMetaByID_CacheCorruptJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	mockCache.On("Get", ctx, "doc1").Return(`{"bad json"`, nil)

	res, err := service.documentMetaByID(ctx, "doc1")

	assert.Nil(t, res)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestDocumentMetaByID_CacheMiss_DBSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{ID: "doc1", Name: "fresh"}

	mockCache.On("Get", ctx, "doc1").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc1").Return(doc, nil)
	mockCache.On("Set", ctx, "doc1", mock.Anything).Return(nil)

	res, err := service.documentMetaByID(ctx, "doc1")

	assert.NoError(t, err)
	assert.Equal(t, doc.ID, res.ID)
	mockRepo.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestDocumentMetaByID_DB_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	mockCache.On("Get", ctx, "doc1").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc1").Return((*models.Document)(nil), models.ErrDocumentNotFound)

	res, err := service.documentMetaByID(ctx, "doc1")

	assert.Nil(t, res)
	assert.ErrorIs(t, err, models.ErrDocumentNotFound)
}

func TestDocumentMetaByID_DB_OtherError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	mockCache.On("Get", ctx, "doc1").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc1").Return((*models.Document)(nil), errors.New("db down"))

	res, err := service.documentMetaByID(ctx, "doc1")

	assert.Nil(t, res)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestDocumentByID_Success_WithoutFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc1",
		Name:    "doc",
		OwnerID: "u1",
		IsFile:  false,
		Grants:  []string{"u1"},
	}
	user := &models.User{ID: "u1", Login: "u1"}

	mockCache.On("Get", ctx, "doc1").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc1").Return(doc, nil)
	mockCache.On("Set", ctx, "doc1", mock.Anything).Return(nil)

	result, file, err := service.DocumentByID(ctx, "doc1", user)

	assert.NoError(t, err)
	assert.Equal(t, doc, result)
	assert.Nil(t, file)
}

func TestDocumentByID_Success_WithFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc2",
		Name:    "file.txt",
		OwnerID: "u1",
		IsFile:  true,
		Grants:  []string{"u1"},
	}
	user := &models.User{ID: "u1", Login: "u1"}

	mockCache.On("Get", ctx, "doc2").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc2").Return(doc, nil)
	mockCache.On("Set", ctx, "doc2", mock.Anything).Return(nil)
	mockStorage.On("LoadFile", doc).Return(io.NopCloser(bytes.NewReader([]byte("file"))), nil)

	result, file, err := service.DocumentByID(ctx, "doc2", user)

	assert.NoError(t, err)
	assert.Equal(t, doc, result)
	assert.NotNil(t, file)
	defer file.Close()
	content, _ := io.ReadAll(file)
	assert.Equal(t, "file", string(content))
}

func TestDocumentByID_documentMetaByID_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	mockCache.On("Get", ctx, "doc404").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc404").Return((*models.Document)(nil), models.ErrDocumentNotFound)

	user := &models.User{ID: "u1", Login: "u1"}
	doc, file, err := service.DocumentByID(ctx, "doc404", user)

	assert.Nil(t, doc)
	assert.Nil(t, file)
	assert.ErrorIs(t, err, models.ErrDocumentNotFound)
}

func TestDocumentByID_Forbidden(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc3",
		Name:    "restricted",
		OwnerID: "owner",
		IsFile:  false,
		Grants:  []string{},
	}
	user := &models.User{ID: "u2", Login: "u2"}

	mockCache.On("Get", ctx, "doc3").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc3").Return(doc, nil)
	mockCache.On("Set", ctx, "doc3", mock.Anything).Return(nil)

	result, file, err := service.DocumentByID(ctx, "doc3", user)

	assert.Nil(t, result)
	assert.Nil(t, file)
	assert.ErrorIs(t, err, models.ErrForbidden)
}

func TestDocumentByID_LoadFileFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc4",
		Name:    "broken",
		OwnerID: "u1",
		IsFile:  true,
		Grants:  []string{"u1"},
	}
	user := &models.User{ID: "u1", Login: "u1"}

	mockCache.On("Get", ctx, "doc4").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc4").Return(doc, nil)
	mockCache.On("Set", ctx, "doc4", mock.Anything).Return(nil)
	mockFile := new(os.File)

	defer mockFile.Close()
	mockStorage.On("LoadFile", doc).Return(mockFile, errors.New("disk fail"))

	result, file, err := service.DocumentByID(ctx, "doc4", user)

	assert.Nil(t, result)
	assert.Nil(t, file)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestDeleteDocument_Success_WithoutFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc1",
		OwnerID: "user1",
		IsFile:  false,
	}
	user := &models.User{ID: "user1", Login: "user1"}

	mockCache.On("Get", ctx, "doc1").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc1").Return(doc, nil)
	mockRepo.On("Delete", ctx, "doc1").Return(nil)
	mockCache.On("Del", ctx, []string{doc.ID, "docs:user1"}).Return(nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything).Return(nil)

	err := service.DeleteDocument(ctx, "doc1", user)
	assert.NoError(t, err)
}

func TestDeleteDocument_Success_WithFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc2",
		OwnerID: "user1",
		IsFile:  true,
	}
	user := &models.User{ID: "user1", Login: "user1"}

	mockCache.On("Get", ctx, "doc2").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc2").Return(doc, nil)
	mockRepo.On("Delete", ctx, "doc2").Return(nil)
	mockCache.On("Del", ctx, []string{doc.ID, "docs:user1"}).Return(nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything).Return(nil)
	mockStorage.On("DeleteFile", doc).Return(nil)

	err := service.DeleteDocument(ctx, "doc2", user)
	assert.NoError(t, err)
}

func TestDeleteDocument_Forbidden(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc3",
		OwnerID: "owner",
		IsFile:  false,
	}
	user := &models.User{ID: "someone_else", Login: "x"}

	mockCache.On("Get", ctx, "doc3").Return("", errors.New("miss"))
	mockCache.On("Set", ctx, mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("DocumentByID", ctx, "doc3").Return(doc, nil)

	err := service.DeleteDocument(ctx, "doc3", user)
	assert.ErrorIs(t, err, models.ErrForbidden)
}

func TestDeleteDocument_MetadataError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u", Login: "u"}

	mockCache.On("Get", ctx, "docX").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "docX").Return((*models.Document)(nil), models.ErrDocumentNotFound)

	err := service.DeleteDocument(ctx, "docX", user)
	assert.ErrorIs(t, err, models.ErrDocumentNotFound)
}

func TestDeleteDocument_DeleteMetaFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{ID: "doc4", OwnerID: "u", IsFile: false}
	user := &models.User{ID: "u", Login: "u"}

	mockCache.On("Get", ctx, "doc4").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc4").Return(doc, nil)
	mockRepo.On("Delete", ctx, "doc4").Return(errors.New("db fail"))
	mockCache.On("Set", ctx, mock.Anything, mock.Anything).Return(nil)

	err := service.DeleteDocument(ctx, "doc4", user)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestDeleteDocument_DeleteMetaNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{ID: "doc5", OwnerID: "u", IsFile: false}
	user := &models.User{ID: "u", Login: "u"}

	mockCache.On("Get", ctx, "doc5").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc5").Return(doc, nil)
	mockRepo.On("Delete", ctx, "doc5").Return(models.ErrDocumentNotFound)
	mockCache.On("Del", ctx, []string{doc.ID, "docs:u"}).Return(nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything).Return(nil)

	err := service.DeleteDocument(ctx, "doc5", user)
	assert.NoError(t, err)
}

func TestDeleteDocument_FileStorageError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{ID: "doc6", OwnerID: "u", IsFile: true}
	user := &models.User{ID: "u", Login: "u"}

	mockCache.On("Get", ctx, "doc6").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc6").Return(doc, nil)
	mockRepo.On("Delete", ctx, "doc6").Return(nil)
	mockCache.On("Del", ctx, []string{doc.ID, "docs:u"}).Return(nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything).Return(nil)
	mockStorage.On("DeleteFile", doc).Return(errors.New("disk fail"))

	err := service.DeleteDocument(ctx, "doc6", user)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestDeleteDocument_FileNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{ID: "doc7", OwnerID: "u", IsFile: true}
	user := &models.User{ID: "u", Login: "u"}

	mockCache.On("Get", ctx, "doc7").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc7").Return(doc, nil)
	mockRepo.On("Delete", ctx, "doc7").Return(nil)
	mockCache.On("Del", ctx, []string{doc.ID, "docs:u"}).Return(nil)
	mockCache.On("Set", ctx, mock.Anything, mock.Anything).Return(nil)
	mockStorage.On("DeleteFile", doc).Return(models.ErrDocumentNotFound)

	err := service.DeleteDocument(ctx, "doc7", user)
	assert.ErrorIs(t, err, models.ErrDocumentNotFound)
}

func TestGrantAccess_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc123",
		OwnerID: "user1",
		Grants:  []string{"u1", "u2"},
	}
	user := &models.User{ID: "user1", Login: "user1"}
	grants := []string{"u1", "u2"}

	// mocks
	mockCache.On("Get", ctx, "doc123").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc123").Return(doc, nil)
	mockCache.On("Set", ctx, "doc123", mock.Anything).Return(nil)

	mockRepo.On("GrantAccess", ctx, "doc123", grants).Return(nil)
	mockCache.On("Del", ctx, []string{doc.ID, "docs:user1"}).Return(nil)
	mockCache.On("Del", ctx, []string{"docs:u1"}).Return(nil)
	mockCache.On("Del", ctx, []string{"docs:u2"}).Return(nil)

	err := service.GrantAccess(ctx, "doc123", user, grants)
	assert.NoError(t, err)
}

func TestGrantAccess_documentMetaByID_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "user1", Login: "user1"}

	mockCache.On("Get", ctx, "doc404").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc404").Return((*models.Document)(nil), models.ErrDocumentNotFound)

	err := service.GrantAccess(ctx, "doc404", user, []string{"a"})
	assert.ErrorIs(t, err, models.ErrDocumentNotFound)
}

func TestGrantAccess_Forbidden(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc123",
		OwnerID: "owner",
		Grants:  []string{},
	}
	user := &models.User{ID: "someone_else", Login: "x"}

	mockCache.On("Get", ctx, "doc123").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc123").Return(doc, nil)
	mockCache.On("Set", ctx, "doc123", mock.Anything).Return(nil)

	err := service.GrantAccess(ctx, "doc123", user, []string{"a"})
	assert.ErrorIs(t, err, models.ErrForbidden)
}

func TestGrantAccess_GrantError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc123",
		OwnerID: "user1",
		Grants:  []string{"a"},
	}
	user := &models.User{ID: "user1", Login: "user1"}

	mockCache.On("Get", ctx, "doc123").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc123").Return(doc, nil)
	mockCache.On("Set", ctx, "doc123", mock.Anything).Return(nil)

	mockRepo.On("GrantAccess", ctx, "doc123", mock.Anything).Return(errors.New("db fail"))

	err := service.GrantAccess(ctx, "doc123", user, []string{"a"})
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestGrantAccess_CacheDelFails_Ignored(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)
	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	doc := &models.Document{
		ID:      "doc123",
		OwnerID: "user1",
		Grants:  []string{"g1", "g2"},
	}
	user := &models.User{ID: "user1", Login: "user1"}
	grants := []string{"g1", "g2"}

	mockCache.On("Get", ctx, "doc123").Return("", errors.New("miss"))
	mockRepo.On("DocumentByID", ctx, "doc123").Return(doc, nil)
	mockCache.On("Set", ctx, "doc123", mock.Anything).Return(nil)

	mockRepo.On("GrantAccess", ctx, "doc123", grants).Return(nil)
	mockCache.On("Del", ctx, []string{doc.ID, "docs:user1"}).Return(errors.New("cache fail"))
	mockCache.On("Del", ctx, []string{"docs:g1"}).Return(errors.New("cache fail"))
	mockCache.On("Del", ctx, []string{"docs:g2"}).Return(errors.New("cache fail"))

	err := service.GrantAccess(ctx, "doc123", user, grants)
	assert.NoError(t, err)
}

func TestListDocuments_CacheHit_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)

	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u1", Login: "u1"}
	filter := models.DocumentFilter{Key: "mime", Value: "text", Limit: 10}

	docs := []*models.Document{{ID: "d1", OwnerID: "u1"}}
	docsJSON, _ := docsToJSON(docs)

	mockCache.On("Get", ctx, "docs:u1:u1:mime:text:10").Return(docsJSON, nil)

	result, err := service.ListDocuments(ctx, user, "u1", filter)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "d1", result[0].ID)
}

func TestListDocuments_CacheMiss_InvalidFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)

	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u1", Login: "u1"}
	filter := models.DocumentFilter{
		Key:   "",
		Value: "invalid value",
	}

	mockCache.On("Get", ctx, "docs:u1:u1::invalid value:0").Return("", errors.New("miss"))

	result, err := service.ListDocuments(ctx, user, "u1", filter)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, models.ErrInvalidParams)
}

func TestListDocuments_RepoSuccess_SetCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)

	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u1", Login: "u1"}
	filter := models.DocumentFilter{Key: "mime", Value: "text", Limit: 1}

	docs := []*models.Document{{ID: "d1", OwnerID: "u1"}}
	docsJSON, _ := docsToJSON(docs)

	mockCache.On("Get", ctx, "docs:u1:u1:mime:text:1").Return("", errors.New("miss"))
	mockRepo.On("FilteredDocuments", ctx, "u1", "u1", filter).Return(docs, nil)
	mockCache.On("Set", ctx, "docs:u1:u1:mime:text:1", docsJSON).Return(nil)

	result, err := service.ListDocuments(ctx, user, "u1", filter)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListDocuments_DocNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)

	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u1", Login: "u1"}
	filter := models.DocumentFilter{Key: "mime", Value: "text", Limit: 1}

	mockCache.On("Get", ctx, "docs:u1:u1:mime:text:1").Return("", errors.New("miss"))
	mockRepo.On("FilteredDocuments", ctx, "u1", "u1", filter).Return([]*models.Document{}, models.ErrDocumentNotFound)

	result, err := service.ListDocuments(ctx, user, "u1", filter)
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestListDocuments_RepoError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)

	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u1", Login: "u1"}
	filter := models.DocumentFilter{Key: "mime", Value: "text", Limit: 1}

	mockCache.On("Get", ctx, "docs:u1:u1:mime:text:1").Return("", errors.New("miss"))
	mockRepo.On("FilteredDocuments", ctx, "u1", "u1", filter).Return([]*models.Document{}, errors.New("db fail"))

	result, err := service.ListDocuments(ctx, user, "u1", filter)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, models.ErrInternal)
}

func TestListDocuments_docsToJSON_ErrorIgnored(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)

	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u1", Login: "u1"}
	filter := models.DocumentFilter{Key: "mime", Value: "text", Limit: 1}

	docs := []*models.Document{{ID: "d1", OwnerID: "u1", JSONData: []byte{0xff}}} // non-UTF8 = JSON still ok

	mockCache.On("Get", ctx, "docs:u1:u1:mime:text:1").Return("", errors.New("miss"))
	mockRepo.On("FilteredDocuments", ctx, "u1", "u1", filter).Return(docs, nil)
	mockCache.On("Set", ctx, "docs:u1:u1:mime:text:1", mock.Anything).Return(errors.New("fail"))

	result, err := service.ListDocuments(ctx, user, "u1", filter)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestListDocuments_CacheInvalidJSON(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mockRepo := new(MockDocumentRepository)
	mockStorage := new(MockFileStorage)
	mockCache := new(MockCache)

	service := New(slog.Default(), mockRepo, mockCache, mockStorage)

	user := &models.User{ID: "u1", Login: "u1"}
	filter := models.DocumentFilter{Key: "mime", Value: "text", Limit: 1}

	mockCache.On("Get", ctx, "docs:u1:u1:mime:text:1").Return("bad-json", nil)

	result, err := service.ListDocuments(ctx, user, "u1", filter)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, models.ErrInternal)
}
