package documentrepo

import (
	"context"
	"database/sql"
	"errors"
	"fileserver/internal/models"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setup(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock, *repository) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)
	return sqlxDB, mock, repo
}

func TestCreateDocument_Success(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	doc := &models.Document{
		ID:        "doc123",
		OwnerID:   "user1",
		Name:      "report.pdf",
		Mime:      "application/pdf",
		IsFile:    true,
		IsPublic:  false,
		Path:      "/docs/report.pdf",
		JSONData:  []byte(`{"author":"nikita"}`),
		CreatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO documents (id, owner_id, name, mime, is_file, is_public, path, json_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)).
		WithArgs(doc.ID, doc.OwnerID, doc.Name, doc.Mime, doc.IsFile, doc.IsPublic, doc.Path, doc.JSONData, doc.CreatedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateDocument(context.Background(), doc)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateDocument_InsertError(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	doc := &models.Document{
		ID:        "doc-error",
		OwnerID:   "userX",
		Name:      "crash.txt",
		Mime:      "text/plain",
		IsFile:    false,
		IsPublic:  true,
		Path:      "/broken/path.txt",
		CreatedAt: time.Now(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO documents (id, owner_id, name, mime, is_file, is_public, path, json_data, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`)).
		WithArgs(doc.ID, doc.OwnerID, doc.Name, doc.Mime, doc.IsFile, doc.IsPublic, doc.Path, nil, doc.CreatedAt).
		WillReturnError(errors.New("db failure"))

	err := repo.CreateDocument(context.Background(), doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CreateDocument")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentByID_Success(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc123"
	createdAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
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
			WHERE d.id = $1`)).
		WithArgs(docID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "owner_id", "name", "mime", "is_file", "is_public", "path", "json_data", "created_at",
		}).AddRow(
			docID, "user1", "file.txt", "text/plain", true, false, "/some/path", []byte(`{"key":"value"}`), createdAt,
		))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			u.login
		FROM grants g
		INNER JOIN users u ON g.user_id = u.id
		WHERE g.document_id = $1`)).
		WithArgs(docID).
		WillReturnRows(sqlmock.NewRows([]string{"login"}).AddRow("grant_user1").AddRow("grant_user2"))

	doc, err := repo.DocumentByID(context.Background(), docID)
	assert.NoError(t, err)
	assert.Equal(t, docID, doc.ID)
	assert.Equal(t, "file.txt", doc.Name)
	assert.Equal(t, []string{"grant_user1", "grant_user2"}, doc.Grants)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentByID_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "not_found"

	mock.ExpectQuery("SELECT.+FROM documents").
		WithArgs(docID).
		WillReturnError(sql.ErrNoRows)

	doc, err := repo.DocumentByID(context.Background(), docID)
	assert.Nil(t, doc)
	assert.ErrorIs(t, err, models.ErrDocumentNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentByID_GrantQueryFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc-with-bad-grants"
	createdAt := time.Now()

	mock.ExpectQuery("SELECT.+FROM documents").
		WithArgs(docID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "owner_id", "name", "mime", "is_file", "is_public", "path", "json_data", "created_at",
		}).AddRow(docID, "user2", "doc.txt", "text/plain", false, false, "/x", nil, createdAt))

	mock.ExpectQuery("SELECT.+FROM grants").
		WithArgs(docID).
		WillReturnError(errors.New("db grant failure"))

	doc, err := repo.DocumentByID(context.Background(), docID)
	assert.Nil(t, doc)
	assert.ErrorContains(t, err, "DocumentByID")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListByUser_Success(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	userID := "user-123"
	createdAt := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
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
		WHERE d.owner_id = $1`)).
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "owner_id", "name", "mime", "is_file", "is_public", "path", "json_data", "created_at",
		}).AddRow(
			"doc1", userID, "doc.txt", "text/plain", true, false, "/some/path", []byte(`{"foo":"bar"}`), createdAt,
		))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT
			u.login
		FROM grants g
		INNER JOIN users u ON g.user_id = u.id
		WHERE g.document_id = $1`)).
		WithArgs("doc1").
		WillReturnRows(sqlmock.NewRows([]string{"login"}).AddRow("grant1"))

	docs, err := repo.ListByUser(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].ID)
	assert.Equal(t, []string{"grant1"}, docs[0].Grants)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListByUser_GrantQueryFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	userID := "user-err"
	createdAt := time.Now()

	mock.ExpectQuery("SELECT.+FROM documents").
		WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "owner_id", "name", "mime", "is_file", "is_public", "path", "json_data", "created_at",
		}).AddRow(
			"doc1", userID, "doc.txt", "text/plain", true, false, "/some/path", []byte(`{}`), createdAt,
		))

	mock.ExpectQuery("SELECT.+FROM grants").
		WithArgs("doc1").
		WillReturnError(errors.New("grant fetch error"))

	docs, err := repo.ListByUser(context.Background(), userID)
	assert.Nil(t, docs)
	assert.ErrorContains(t, err, "ListByUser")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListByUser_DBQueryFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	userID := "user-fail"

	mock.ExpectQuery("SELECT.+FROM documents").
		WithArgs(userID).
		WillReturnError(errors.New("db fail"))

	docs, err := repo.ListByUser(context.Background(), userID)
	assert.Nil(t, docs)
	assert.ErrorContains(t, err, "ListByUser")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc123"

	mock.ExpectExec(`DELETE FROM documents WHERE id = \$1`).
		WithArgs(docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.Delete(context.Background(), docID)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDelete_Failure(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc456"

	mock.ExpectExec(`DELETE FROM documents WHERE id = \$1`).
		WithArgs(docID).
		WillReturnError(errors.New("db error"))

	err := repo.Delete(context.Background(), docID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "documentRepo/Delete")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGrantAccess_Success(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc123"
	userIDs := []string{"u1", "u2"}

	mock.ExpectBegin()

	mock.ExpectExec(`DELETE FROM grants WHERE document_id = \$1`).
		WithArgs(docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	for _, uID := range userIDs {
		mock.ExpectExec(`INSERT INTO grants \(document_id, user_id\) VALUES \(\$1, \$2\)`).
			WithArgs(docID, uID).
			WillReturnResult(sqlmock.NewResult(0, 1))
	}

	mock.ExpectCommit()

	err := repo.GrantAccess(context.Background(), docID, userIDs)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGrantAccess_BeginTxFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc123"
	userIDs := []string{"u1"}

	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))

	err := repo.GrantAccess(context.Background(), docID, userIDs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GrantAccess")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGrantAccess_DeleteFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc123"
	userIDs := []string{"u1"}

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM grants WHERE document_id = \$1`).
		WithArgs(docID).
		WillReturnError(errors.New("delete failed"))

	err := repo.GrantAccess(context.Background(), docID, userIDs)
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGrantAccess_InsertFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc123"
	userIDs := []string{"u1", "u2"}

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM grants WHERE document_id = \$1`).
		WithArgs(docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`INSERT INTO grants \(document_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(docID, "u1").
		WillReturnError(errors.New("insert failed"))

	err := repo.GrantAccess(context.Background(), docID, userIDs)
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGrantAccess_CommitFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	docID := "doc123"
	userIDs := []string{"u1"}

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM grants WHERE document_id = \$1`).
		WithArgs(docID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec(`INSERT INTO grants \(document_id, user_id\) VALUES \(\$1, \$2\)`).
		WithArgs(docID, "u1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	err := repo.GrantAccess(context.Background(), docID, userIDs)
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentsGrantedTo_Success(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	userID := "user1"

	rows := sqlmock.NewRows([]string{"id", "owner_id", "name", "mime", "is_file", "is_public", "path", "json_data", "created_at"}).
		AddRow("doc1", "owner1", "name.txt", "text/plain", true, false, "/some/path", nil, time.Now())

	mock.ExpectQuery(`SELECT (.+) FROM documents d INNER JOIN grants g ON g.document_id = d.id WHERE g.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(rows)

	grantRows := sqlmock.NewRows([]string{"login"}).AddRow("grantee1").AddRow("grantee2")
	mock.ExpectQuery(`SELECT u.login FROM grants g INNER JOIN users u ON g.user_id = u.id WHERE g.document_id = \$1`).
		WithArgs("doc1").
		WillReturnRows(grantRows)

	docs, err := repo.DocumentsGrantedTo(context.Background(), userID)
	assert.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].ID)
	assert.ElementsMatch(t, []string{"grantee1", "grantee2"}, docs[0].Grants)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentsGrantedTo_SelectFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	userID := "user1"

	mock.ExpectQuery(`SELECT (.+) FROM documents d INNER JOIN grants g ON g.document_id = d.id WHERE g.user_id = \$1`).
		WithArgs(userID).
		WillReturnError(errors.New("select failed"))

	docs, err := repo.DocumentsGrantedTo(context.Background(), userID)
	assert.Error(t, err)
	assert.Nil(t, docs)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDocumentsGrantedTo_GetGrantsFails(t *testing.T) {
	t.Parallel()

	db, mock, repo := setup(t)
	defer db.Close()

	userID := "user1"

	rows := sqlmock.NewRows([]string{"id", "owner_id", "name", "mime", "is_file", "is_public", "path", "json_data", "created_at"}).
		AddRow("doc1", "owner1", "name.txt", "text/plain", true, false, "/some/path", nil, time.Now())

	mock.ExpectQuery(`SELECT (.+) FROM documents d INNER JOIN grants g ON g.document_id = d.id WHERE g.user_id = \$1`).
		WithArgs(userID).
		WillReturnRows(rows)

	mock.ExpectQuery(`SELECT u.login FROM grants g INNER JOIN users u ON g.user_id = u.id WHERE g.document_id = \$1`).
		WithArgs("doc1").
		WillReturnError(errors.New("getGrants failed"))

	docs, err := repo.DocumentsGrantedTo(context.Background(), userID)
	assert.Error(t, err)
	assert.Nil(t, docs)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFilteredDocuments_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	filter := models.DocumentFilter{
		Key:   "mime",
		Value: "image/png",
		Limit: 10,
	}
	requesterID := "user-123"
	login := "owner"

	rows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "mime", "is_file", "is_public",
		"path", "json_data", "created_at",
	}).AddRow("doc-1", "user-123", "example.png", "image/png", true, true, "/path", `{"tag":"png"}`, time.Now())

	mock.ExpectQuery("SELECT d.id AS id").
		WithArgs("mime", "image/png", requesterID, login, 10).
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT u.login").
		WithArgs("doc-1").
		WillReturnRows(sqlmock.NewRows([]string{"login"}).AddRow("user-123"))

	docs, err := repo.FilteredDocuments(context.Background(), login, requesterID, filter)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "doc-1", docs[0].ID)
}

func TestFilteredDocuments_NoRows(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	filter := models.DocumentFilter{Key: "mime", Value: "text/plain"}
	mock.ExpectQuery("SELECT d.id AS id").
		WithArgs("mime", "text/plain", "req-1", "", sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	docs, err := repo.FilteredDocuments(context.Background(), "", "req-1", filter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FilteredOrders")
	assert.Nil(t, docs)
}

func TestFilteredDocuments_GrantsError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	filter := models.DocumentFilter{}

	rows := sqlmock.NewRows([]string{
		"id", "owner_id", "name", "mime", "is_file", "is_public",
		"path", "json_data", "created_at",
	}).AddRow("doc-2", "user-456", "example.txt", "text/plain", false, false, "/path", nil, time.Now())

	mock.ExpectQuery("SELECT d.id AS id").
		WithArgs("", "", "req-2", "", sqlmock.AnyArg()).
		WillReturnRows(rows)

	mock.ExpectQuery("SELECT u.login").
		WithArgs("doc-2").
		WillReturnError(errors.New("db failure"))

	docs, err := repo.FilteredDocuments(context.Background(), "", "req-2", filter)
	require.Error(t, err)
	assert.Nil(t, docs)
}

func TestGetGrants_Success(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	docID := "doc-123"
	mock.ExpectQuery(`SELECT\s+u\.login`).
		WithArgs(docID).
		WillReturnRows(sqlmock.NewRows([]string{"login"}).
			AddRow("user1").
			AddRow("user2"))

	grants, err := repo.getGrants(context.Background(), docID)
	require.NoError(t, err)
	require.Len(t, grants, 2)
	assert.Equal(t, []string{"user1", "user2"}, grants)
}

func TestGetGrants_DBError(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := NewRepository(sqlxDB)

	docID := "doc-456"
	mock.ExpectQuery(`SELECT\s+u\.login`).
		WithArgs(docID).
		WillReturnError(errors.New("db failure"))

	grants, err := repo.getGrants(context.Background(), docID)
	require.Error(t, err)
	assert.Nil(t, grants)
	assert.Contains(t, err.Error(), "getGrants")
}
