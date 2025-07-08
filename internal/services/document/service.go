package documentservice

import (
	"context"
	"encoding/json"
	"errors"
	"fileserver/internal/models"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"

	uuid "github.com/satori/go.uuid"
)

const pkg = "documentService/"

type DocumentService struct {
	log         *slog.Logger
	docRepo     DocumentRepository
	cache       Cache
	fileStorage FileStorage
}

func New(
	log *slog.Logger,
	docRepo DocumentRepository,
	cache Cache,
	fileStorage FileStorage,
) *DocumentService {
	return &DocumentService{
		log:         log,
		docRepo:     docRepo,
		cache:       cache,
		fileStorage: fileStorage,
	}
}

func (ds *DocumentService) UploadDocument(ctx context.Context, requester *models.User, doc *models.Document, content io.Reader) (string, error) {
	op := pkg + "UploadDocument"

	log := ds.log.With(slog.String("op", op))

	log.Debug("attempting to upload document", slog.String("name", doc.Name), slog.Bool("is_file", doc.IsFile))

	doc.ID = uuid.NewV4().String()
	doc.CreatedAt = time.Now()

	if doc.IsFile {
		err := ds.fileStorage.SaveFile(doc, content)
		if err != nil {
			log.Error("failed to save file", slog.String("error", err.Error()))
			return "", fmt.Errorf("%s: %w", op, models.ErrInternal)
		}
	}

	err := ds.docRepo.CreateDocument(ctx, doc)
	if err != nil {
		log.Error("failed to save document metadata", slog.String("error", err.Error()))
		if doc.IsFile {
			_ = ds.fileStorage.DeleteFile(doc)
		}

		return "", fmt.Errorf("%s: %w", op, models.ErrInternal)
	}

	if len(doc.Grants) > 0 {
		err = ds.docRepo.GrantAccess(ctx, doc.ID, doc.Grants)
		if err != nil {
			log.Error("failed to save grants", slog.String("error", err.Error()))
			_ = ds.docRepo.Delete(ctx, doc.ID)
			if doc.IsFile {
				_ = ds.fileStorage.DeleteFile(doc)
			}
			return "", fmt.Errorf("%s: %w", op, models.ErrInternal)
		}
		for _, grant := range doc.Grants {
			if err := ds.cache.Del(ctx, fmt.Sprintf("docs:%s", grant)); err != nil {
				log.Error("failed to invalidate grant cache", slog.String("error", err.Error()))
			}
		}
	}

	if err = ds.cache.Del(ctx, fmt.Sprintf("docs:%s", requester.Login)); err != nil {
		log.Error("failed to invalidate owner cache", slog.String("error", err.Error()))
	}

	log.Debug("document uploaded successfully", slog.String("doc_id", doc.ID), slog.String("owner_id", doc.OwnerID))

	return doc.ID, nil
}

func (ds *DocumentService) DocumentByID(ctx context.Context, docID string, requester *models.User) (*models.Document, io.ReadCloser, error) {
	op := pkg + "DocumentByID"

	log := ds.log.With(slog.String("op", op))

	log.Debug("attempting to get document by id", slog.String("doc_id", docID), slog.String("user_id", requester.ID))

	doc, err := ds.documentMetaByID(ctx, docID)
	if err != nil {
		return nil, nil, err
	}

	if !hasReadAccess(doc, requester.ID, requester.Login) {
		log.Warn("user doesn't have access for document", slog.String("doc_id", docID), slog.String("user_id", requester.ID))
		return nil, nil, models.ErrForbidden
	}

	var file io.ReadCloser

	if doc.IsFile {
		file, err = ds.fileStorage.LoadFile(doc)
		if err != nil {
			log.Error("failed to load file from storage", slog.String("error", err.Error()))
			return nil, nil, models.ErrInternal
		}
	}

	log.Debug("document with content found successfully", slog.String("doc_id", docID))

	return doc, file, nil
}

func (ds *DocumentService) DeleteDocument(ctx context.Context, docID string, requester *models.User) error {
	op := pkg + "DeleteDocument"

	log := ds.log.With(slog.String("op", op))

	log.Debug("attempting to delete document", slog.String("doc_id", docID), slog.String("user_id", requester.ID))

	doc, err := ds.documentMetaByID(ctx, docID)
	if err != nil {
		log.Warn("failed to get document by id", slog.String("error", err.Error()))
		return err
	}

	if !hasDeleteAccess(doc, requester.ID) {
		log.Warn("user doesn't have access for delete operation", slog.String("doc_id", docID), slog.String("user_id", requester.ID))
		return models.ErrForbidden
	}

	if err := ds.docRepo.Delete(ctx, docID); err != nil {
		if errors.Is(err, models.ErrDocumentNotFound) {
			log.Warn("failed to delete document meta", slog.String("error", models.ErrDocumentNotFound.Error()))
		} else {
			log.Error("failed to delete document meta", slog.String("error", err.Error()))
			return models.ErrInternal
		}
	}

	cacheKey := fmt.Sprintf("docs:%s", requester.Login)

	err = ds.cache.Del(ctx, doc.ID, cacheKey)
	if err != nil {
		log.Error("failed to delete doc from cache", slog.String("error", err.Error()))
	}

	if doc.IsFile {
		if err := ds.fileStorage.DeleteFile(doc); err != nil {
			if errors.Is(err, models.ErrDocumentNotFound) {
				log.Warn("failed to delete document from file storage", slog.String("error", models.ErrDocumentNotFound.Error()))
				return models.ErrDocumentNotFound
			} else {
				log.Error("failed to delete document content", slog.String("error", err.Error()))
				return models.ErrInternal
			}
		}

	}

	log.Debug("document with content deleted successfully", slog.String("doc_id", docID), slog.String("user_id", requester.ID))

	return nil
}

func (ds *DocumentService) GrantAccess(ctx context.Context, docID string, requester *models.User, grants []string) error {
	op := pkg + "GrantAccess"

	log := ds.log.With(slog.String("op", op))

	log.Debug("attempting to grant access", slog.String("doc_id", docID), slog.String("user_id", requester.ID))

	doc, err := ds.documentMetaByID(ctx, docID)
	if err != nil {
		log.Error("failed to get document by id", slog.String("error", err.Error()))
		return err
	}

	if !hasGrantAccess(doc, requester.ID) {
		log.Warn("user doesn't have permission for grant access", slog.String("doc_id", docID), slog.String("user_id", requester.ID))
		return models.ErrForbidden
	}

	if err := ds.docRepo.GrantAccess(ctx, docID, grants); err != nil {
		log.Error("failed to grant access", slog.String("error", err.Error()))
		return models.ErrInternal
	}

	cacheKey := fmt.Sprintf("docs:%s", requester.Login)

	err = ds.cache.Del(ctx, doc.ID, cacheKey)
	if err != nil {
		log.Error("failed to delete doc from cache", slog.String("error", err.Error()))
	}

	for _, grant := range doc.Grants {
		if err := ds.cache.Del(ctx, fmt.Sprintf("docs:%s", grant)); err != nil {
			log.Error("failed to invalidate grant cache", slog.String("error", err.Error()))
		}
	}

	log.Debug("grant access was successfull", slog.String("doc_id", docID), slog.String("user_id", requester.ID))

	return nil
}

func (ds *DocumentService) ListDocuments(ctx context.Context, requester *models.User, login string, filter models.DocumentFilter) ([]*models.Document, error) {
	op := pkg + "ListDocuments"

	log := ds.log.With(slog.String("op", op))

	log.Debug("attempting to list documents",
		slog.String("requester_id", requester.ID),
		slog.String("owner_login", login),
		slog.String("filter_key", filter.Key),
		slog.String("filter_value", filter.Value),
		slog.Int("limit", filter.Limit))

	var docs []*models.Document

	cacheKey := fmt.Sprintf("docs:%s:%s:%s:%s:%v", requester.Login, login, filter.Key, filter.Value, filter.Limit)

	docsJSON, err := ds.cache.Get(ctx, cacheKey)
	if err != nil || docsJSON == "" { // TODO: refactor
		log.Warn("failed to get docs from cache")

		if !filter.IsValid() {
			log.Warn("invalid filter format")
			return nil, models.ErrInvalidParams
		}

		docs, err = ds.docRepo.FilteredDocuments(ctx, login, requester.ID, filter)
		if err != nil {
			if errors.Is(err, models.ErrDocumentNotFound) {
				return nil, nil
			}
			log.Error("failed to list documents", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: %w", op, models.ErrInternal)
		}

		docsJSON, err = docsToJSON(docs)
		if err != nil {
			log.Error("failed to convert docs to json", slog.String("error", err.Error()))
		} else {
			err = ds.cache.Set(ctx, cacheKey, docsJSON)
			if err != nil {
				log.Error("failed to set docs in cache", slog.String("error", err.Error()))
			}
		}

		return docs, nil
	} else {
		docs, err = jsonToDocs(docsJSON)
		if err != nil {
			log.Error("failed to parse json to docs", slog.String("error", err.Error()))
			return nil, models.ErrInternal
		}
	}

	log.Debug("documents listed successfully",
		slog.Int("count", len(docs)),
		slog.String("requester_id", requester.ID))

	return docs, nil
}

func (ds *DocumentService) documentMetaByID(ctx context.Context, docID string) (*models.Document, error) {
	op := pkg + "documentMetaByID"

	log := ds.log.With(slog.String("op", op))

	var doc *models.Document

	docJSON, err := ds.cache.Get(ctx, docID)
	if err != nil || docJSON == "" { // TODO: refactor
		log.Debug("failed to get doc in cache by id", slog.String("doc_id", docID))

		doc, err = ds.docRepo.DocumentByID(ctx, docID)
		if err != nil {
			if errors.Is(err, models.ErrDocumentNotFound) {
				log.Warn("document not found", slog.String("doc_id", docID))
				return nil, fmt.Errorf("%s: %w", op, models.ErrDocumentNotFound)
			}
			log.Error("failed to get document by id", slog.String("error", err.Error()))
			return nil, fmt.Errorf("%s: %w", op, models.ErrInternal)
		}

		docJSON, err := docToJSON(doc)
		if err != nil {
			log.Error("failed to parse doc to json", slog.String("error", err.Error()))
		} else {
			err = ds.cache.Set(ctx, docID, docJSON)
			if err != nil {
				log.Warn("failed to set doc to cache", slog.String("error", err.Error()))
			}
		}

		return doc, nil
	} else {
		doc, err = jsonToDoc(docJSON)
		if err != nil {
			log.Error("failed to parse json to doc", slog.String("error", err.Error()))
			return nil, models.ErrInternal
		}
	}

	return doc, nil
}

func jsonToDocs(s string) ([]*models.Document, error) {
	if len(s) == 0 {
		return nil, errors.New("empty json string")
	}
	var docs []*models.Document

	if err := json.Unmarshal([]byte(s), &docs); err != nil {
		return nil, err
	}

	return docs, nil
}

func docsToJSON(docs []*models.Document) (string, error) {
	res, err := json.Marshal(docs)
	if err != nil {
		return "", err
	}

	return string(res), nil
}

func docToJSON(doc *models.Document) (string, error) {
	jsonSlice, err := json.Marshal(doc)
	if err != nil {
		return "", err
	}

	return string(jsonSlice), nil
}

func jsonToDoc(s string) (*models.Document, error) {
	if len(s) == 0 {
		return nil, errors.New("empty json string")
	}

	var doc models.Document
	if err := json.Unmarshal([]byte(s), &doc); err != nil {
		return nil, err
	}

	return &doc, nil
}

func hasReadAccess(doc *models.Document, userID string, userLogin string) bool {
	return doc.IsPublic || doc.OwnerID == userID || slices.Contains(doc.Grants, userLogin)
}

func hasDeleteAccess(doc *models.Document, userID string) bool {
	return doc.OwnerID == userID
}

func hasGrantAccess(doc *models.Document, userID string) bool {
	return doc.OwnerID == userID
}
