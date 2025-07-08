package filerepo

import (
	"errors"
	"fileserver/internal/models"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const pkg = "fileRepo/"

type repository struct {
	path string
}

func NewRepository(path string) *repository {
	return &repository{path: path}
}

func (r *repository) SaveFile(doc *models.Document, reader io.Reader) error {
	op := pkg + "SaveFile"

	ext := filepath.Ext(doc.Name)
	fullPath := filepath.Join(r.path, doc.ID+ext)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	out, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer out.Close()

	_, err = io.Copy(out, reader)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	doc.Path = fullPath
	return err
}

func (r *repository) LoadFile(doc *models.Document) (io.ReadCloser, error) {
	op := pkg + "LoadFile"

	file, err := os.Open(doc.Path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return file, nil
}

func (r *repository) DeleteFile(doc *models.Document) error {
	op := pkg + "DeleteFile"

	err := os.Remove(doc.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return models.ErrDocumentNotFound
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
