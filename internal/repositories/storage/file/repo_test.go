package filerepo

import (
	"bytes"
	"errors"
	"fileserver/internal/models"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSaveLoadDelete(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	repo := NewRepository(tmpDir)

	doc := &models.Document{Name: "test.txt", Path: "testfile.txt"}
	content := []byte("hello world")

	err := repo.SaveFile(doc, bytes.NewReader(content))
	assert.NoError(t, err)

	reader, err := repo.LoadFile(doc)
	assert.NoError(t, err)
	defer reader.Close()

	readContent, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, content, readContent)

	err = reader.Close()
	assert.NoError(t, err)

	err = repo.DeleteFile(doc)
	assert.NoError(t, err)

	_, err = os.Stat(doc.Path)
	assert.True(t, os.IsNotExist(err))
}

func TestLocalFileRepository_LoadFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepository(tmpDir)

	doc := &models.Document{Name: "file.txt", Path: "nofile.txt"}

	_, err := repo.LoadFile(doc)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestLocalFileRepository_DeleteFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewRepository(tmpDir)

	doc := &models.Document{Path: "missing.txt"}

	err := repo.DeleteFile(doc)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, models.ErrDocumentNotFound))
}

func TestSaveFile_MkdirAllFails(t *testing.T) {
	t.Parallel()

	repo := NewRepository("///invalid///path///")
	doc := &models.Document{ID: "fail", Name: "file.txt"}

	err := repo.SaveFile(doc, bytes.NewReader([]byte("data")))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SaveFile")
}

func TestSaveFile_CreateFails(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	conflictPath := filepath.Join(tmpDir, "conflict.txt")
	err := os.Mkdir(conflictPath, 0755)
	assert.NoError(t, err)

	doc := &models.Document{
		ID:   "conflict",
		Name: ".txt",
	}

	repo := NewRepository(tmpDir)
	err = repo.SaveFile(doc, bytes.NewReader([]byte("data")))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SaveFile")
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) {
	return 0, errors.New("copy error")
}

func TestSaveFile_CopyFails(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repo := NewRepository(tmpDir)

	doc := &models.Document{ID: "fail", Name: "copy.txt"}
	err := repo.SaveFile(doc, brokenReader{})

	assert.Error(t, err)
}
