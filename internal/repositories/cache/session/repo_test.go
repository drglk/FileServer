package cachesessionrepo

import (
	"context"
	"errors"
	"fileserver/internal/models"
	cacherepo "fileserver/internal/repositories/cache"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockCache struct {
	mock.Mock
}

type mockResponse[T any] struct {
	mock.Mock
	val T
	err error
}

func (m *mockCache) Get(ctx context.Context, key string) cacherepo.CacheResponse[string] {
	args := m.Called(ctx, key)
	return args.Get(0).(cacherepo.CacheResponse[string])
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) cacherepo.CacheResponse[string] {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(cacherepo.CacheResponse[string])
}

func (m *mockCache) Del(ctx context.Context, keys ...string) cacherepo.CacheResponse[int64] {
	args := m.Called(ctx, keys)
	return args.Get(0).(cacherepo.CacheResponse[int64])
}

func (r *mockResponse[T]) Err() error {
	return r.err
}

func (r *mockResponse[T]) Result() (T, error) {
	return r.val, r.err
}

func TestSaveSession_Success(t *testing.T) {
	t.Parallel()

	mockCache := new(mockCache)
	mockResp := &mockResponse[string]{err: nil}

	mockCache.On("Set", mock.Anything, "token123", "user-data", time.Minute).
		Return(mockResp)

	repo := New(mockCache, time.Minute)

	err := repo.SaveSession(context.Background(), "token123", "user-data")
	assert.NoError(t, err)
}

func TestDeleteSession_Success(t *testing.T) {
	t.Parallel()

	mockCache := new(mockCache)
	mockResp := &mockResponse[int64]{err: nil}

	mockCache.On("Del", mock.Anything, []string{"token123"}).
		Return(mockResp)

	repo := New(mockCache, time.Minute)

	err := repo.DeleteSession(context.Background(), "token123")
	assert.NoError(t, err)
}

func TestGetUserByToken_Success(t *testing.T) {
	t.Parallel()

	mockCache := new(mockCache)
	mockResp := &mockResponse[string]{val: "user-data", err: nil}

	mockCache.On("Get", mock.Anything, "token123").
		Return(mockResp)

	repo := New(mockCache, time.Minute)

	result, err := repo.GetUserByToken(context.Background(), "token123")
	assert.NoError(t, err)
	assert.Equal(t, "user-data", result)
}

func TestGetUserByToken_NotFound(t *testing.T) {
	t.Parallel()

	mockCache := new(mockCache)
	mockResp := &mockResponse[string]{val: "", err: nil}

	mockCache.On("Get", mock.Anything, "invalid").
		Return(mockResp)

	repo := New(mockCache, time.Minute)

	result, err := repo.GetUserByToken(context.Background(), "invalid")
	assert.ErrorIs(t, err, models.ErrSessionNotFound)
	assert.Empty(t, result)
}

func TestGetUserByToken_Error(t *testing.T) {
	t.Parallel()

	mockCache := new(mockCache)
	mockResp := &mockResponse[string]{val: "", err: errors.New("connection error")}

	mockCache.On("Get", mock.Anything, "error-token").
		Return(mockResp)

	repo := New(mockCache, time.Minute)

	result, err := repo.GetUserByToken(context.Background(), "error-token")
	assert.Error(t, err)
	assert.Empty(t, result)
}
