package userrepo

import (
	"context"
	"database/sql"
	"fileserver/internal/models"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestAddUser_Success(t *testing.T) {
	t.Parallel()

	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewRepository(sqlxDB)

	user := models.User{
		ID:       "1",
		Login:    "test",
		PassHash: []byte("hashed"),
	}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Login, user.PassHash).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.AddUser(context.Background(), user)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestAddUser_UniqueViolation(t *testing.T) {
	t.Parallel()

	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewRepository(sqlxDB)

	user := models.User{
		ID:       "1",
		Login:    "test",
		PassHash: []byte("hashed"),
	}

	pqErr := &pq.Error{Code: "23505", Constraint: "users_login_key"}

	mock.ExpectExec("INSERT INTO users").
		WithArgs(user.ID, user.Login, user.PassHash).
		WillReturnError(pqErr)

	err := repo.AddUser(context.Background(), user)
	assert.ErrorIs(t, err, models.ErrUNIQUEConstraintFailed)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserByID_Success(t *testing.T) {
	t.Parallel()

	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewRepository(sqlxDB)

	rows := sqlmock.NewRows([]string{"id", "login", "pass_hash"}).
		AddRow("1", "test", []byte("hashed"))

	mock.ExpectQuery("SELECT(.|\n)*FROM users u WHERE u.id").
		WithArgs("1").
		WillReturnRows(rows)

	user, err := repo.UserByID(context.Background(), "1")
	assert.NoError(t, err)
	assert.Equal(t, "test", user.Login)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserByID_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewRepository(sqlxDB)

	mock.ExpectQuery("SELECT(.|\n)*FROM users u WHERE u.id").
		WithArgs("1").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.UserByID(context.Background(), "1")
	assert.ErrorIs(t, err, models.ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserByLogin_Success(t *testing.T) {
	t.Parallel()

	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewRepository(sqlxDB)

	rows := sqlmock.NewRows([]string{"id", "login", "pass_hash"}).
		AddRow("1", "test", []byte("hashed"))

	mock.ExpectQuery("SELECT(.|\n)*FROM users u WHERE u.login").
		WithArgs("test").
		WillReturnRows(rows)

	user, err := repo.UserByLogin(context.Background(), "test")
	assert.NoError(t, err)
	assert.Equal(t, "test", user.Login)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserByLogin_NotFound(t *testing.T) {
	t.Parallel()

	db, mock, _ := sqlmock.New()
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "sqlmock")
	repo := NewRepository(sqlxDB)

	mock.ExpectQuery("SELECT(.|\n)*FROM users u WHERE u.login").
		WithArgs("test").
		WillReturnError(sql.ErrNoRows)

	_, err := repo.UserByLogin(context.Background(), "test")
	assert.ErrorIs(t, err, models.ErrUserNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}
