package documentrepo

import (
	"context"
	"database/sql"
	"errors"
	"fileserver/internal/entities"
	"fileserver/internal/models"
	"fmt"

	"github.com/jmoiron/sqlx"
)

const pkg = "documentRepo/"

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

func (r *repository) CreateDocument(ctx context.Context, doc *models.Document) error {
	op := pkg + "CreateDocument"

	var jsonData any

	if len(doc.JSONData) > 0 {
		jsonData = doc.JSONData
	} else {
		jsonData = nil
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO documents (id, owner_id, name, mime, is_file, is_public, path, json_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		doc.ID, doc.OwnerID, doc.Name, doc.Mime, doc.IsFile, doc.IsPublic, doc.Path, jsonData, doc.CreatedAt)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *repository) DocumentByID(ctx context.Context, id string) (*models.Document, error) {
	op := pkg + "DocumentByID"

	rawDoc := entities.Document{}

	err := r.db.GetContext(ctx, &rawDoc,
		`SELECT
			d.id AS id,
			d.owner_id AS owner_id,
			d.name AS name,
			d.mime AS mime,
			d.is_file AS is_file,
			d.is_public AS is_public,
			d.path AS path,
			d.json_data AS json_data,
			d.created_at AS created_at
			FROM documents d
			WHERE d.id = $1`,
		id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, models.ErrDocumentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	rawGrants, err := r.getGrants(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &models.Document{
		ID:        rawDoc.ID,
		OwnerID:   rawDoc.OwnerID,
		Name:      rawDoc.Name,
		Mime:      rawDoc.Mime,
		IsFile:    rawDoc.IsFile,
		IsPublic:  rawDoc.IsPublic,
		Path:      rawDoc.Path,
		JSONData:  rawDoc.JSONData,
		Grants:    rawGrants,
		CreatedAt: rawDoc.CreatedAt,
	}, nil
}

func (r *repository) ListByUser(ctx context.Context, userID string) ([]*models.Document, error) {
	op := pkg + "ListByUser"

	rawDocs := make([]entities.Document, 0)

	err := r.db.SelectContext(ctx, &rawDocs,
		`SELECT
			d.id AS id,
			d.owner_id AS owner_id,
			d.name AS name,
			d.mime AS mime,
			d.is_file AS is_file,
			d.is_public AS is_public,
			d.path AS path,
			d.json_data AS json_data,
			d.created_at AS created_at
		FROM documents d
		WHERE d.owner_id = $1`,
		userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, models.ErrDocumentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	docs := make([]*models.Document, 0)

	for _, rawDoc := range rawDocs {
		rawGrants, err := r.getGrants(ctx, rawDoc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		docs = append(docs, &models.Document{
			ID:        rawDoc.ID,
			OwnerID:   rawDoc.OwnerID,
			Name:      rawDoc.Name,
			Mime:      rawDoc.Mime,
			IsFile:    rawDoc.IsFile,
			IsPublic:  rawDoc.IsPublic,
			Path:      rawDoc.Path,
			JSONData:  rawDoc.JSONData,
			Grants:    rawGrants,
			CreatedAt: rawDoc.CreatedAt,
		})
	}

	return docs, nil
}

func (r *repository) Delete(ctx context.Context, id string) error {
	op := pkg + "Delete"

	_, err := r.db.ExecContext(ctx,
		`DELETE FROM documents WHERE id = $1`,
		id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *repository) GrantAccess(ctx context.Context, docID string, userIDs []string) error {
	op := pkg + "GrantAccess"

	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	_, err = tx.ExecContext(ctx,
		`DELETE FROM grants WHERE document_id = $1`, docID)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	for _, uID := range userIDs {
		_, err = tx.ExecContext(ctx,
			`INSERT INTO grants (document_id, user_id) VALUES ($1, $2)`,
			docID, uID)
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil

}

func (r *repository) DocumentsGrantedTo(ctx context.Context, userID string) ([]*models.Document, error) {
	op := pkg + "DocumentsGrantedTo"

	rawDocs := make([]entities.Document, 0)

	err := r.db.SelectContext(ctx, &rawDocs,
		`SELECT
			d.id AS id,
			d.owner_id AS owner_id,
			d.name AS name,
			d.mime AS mime,
			d.is_file AS is_file,
			d.is_public AS is_public,
			d.path AS path,
			d.json_data AS json_data,
			d.created_at AS created_at
			FROM documents d
			INNER JOIN grants g ON g.document_id = d.id
			WHERE g.user_id = $1`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	docs := make([]*models.Document, 0)

	for _, rawDoc := range rawDocs {
		rawGrants, err := r.getGrants(ctx, rawDoc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		docs = append(docs, &models.Document{
			ID:        rawDoc.ID,
			OwnerID:   rawDoc.OwnerID,
			Name:      rawDoc.Name,
			Mime:      rawDoc.Mime,
			IsFile:    rawDoc.IsFile,
			IsPublic:  rawDoc.IsPublic,
			Path:      rawDoc.Path,
			JSONData:  rawDoc.JSONData,
			Grants:    rawGrants,
			CreatedAt: rawDoc.CreatedAt,
		})
	}

	return docs, nil
}

func (r *repository) FilteredDocuments(ctx context.Context, login string, requesterID string, filter models.DocumentFilter) ([]*models.Document, error) {
	op := pkg + "FilteredOrders"

	rawDocs := make([]entities.Document, 0)

	baseQuery := `SELECT
			d.id AS id,
			d.owner_id AS owner_id,
			d.name AS name,
			d.mime AS mime,
			d.is_file AS is_file,
			d.is_public AS is_public,
			d.path AS path,
			d.json_data AS json_data,
			d.created_at AS created_at
		FROM documents d
		INNER JOIN users u ON d.owner_id = u.id
		LEFT JOIN grants g ON g.document_id = d.id
		WHERE (
			d.is_public = TRUE
			OR d.owner_id = $3
			OR g.user_id = $3
		)
		AND (
			($1 = 'name' AND d.name = $2) OR
			($1 = 'mime' AND d.mime = $2) OR
			($1 = '' AND TRUE)
		)
		AND ($4 = '' OR u.login = $4)
		ORDER BY d.name ASC, d.created_at DESC`

	args := []any{
		filter.Key,
		filter.Value,
		requesterID,
		login,
	}

	if filter.Limit > 0 {
		args = append(args, filter.Limit)

		baseQuery += ` LIMIT $5`
	}

	err := r.db.SelectContext(ctx, &rawDocs, baseQuery, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w", op, models.ErrDocumentNotFound)
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	docs := make([]*models.Document, 0)

	for _, rawDoc := range rawDocs {
		rawGrants, err := r.getGrants(ctx, rawDoc.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		docs = append(docs, &models.Document{
			ID:        rawDoc.ID,
			OwnerID:   rawDoc.OwnerID,
			Name:      rawDoc.Name,
			Mime:      rawDoc.Mime,
			IsFile:    rawDoc.IsFile,
			IsPublic:  rawDoc.IsPublic,
			Path:      rawDoc.Path,
			JSONData:  rawDoc.JSONData,
			Grants:    rawGrants,
			CreatedAt: rawDoc.CreatedAt,
		})
	}

	return docs, nil
}

func (r *repository) getGrants(ctx context.Context, docID string) ([]string, error) {
	op := pkg + "getGrants"

	logins := make([]string, 0)

	err := r.db.SelectContext(ctx, &logins,
		`SELECT
			u.login
		FROM grants g
		INNER JOIN users u ON g.user_id = u.id
		WHERE g.document_id = $1`,
		docID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return logins, nil
}
