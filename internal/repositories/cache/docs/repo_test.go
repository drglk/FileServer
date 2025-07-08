package cachedocsrepo

import (
	"context"
	"errors"
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
	args := r.Called()
	return args.Error(0)
}

func (r *mockResponse[T]) Result() (T, error) {
	args := r.Called()
	return args.Get(0).(T), args.Error(1)
}

func TestCachedDocsRepo_Set_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := new(mockCache)
	resp := new(mockResponse[string])
	repo := New(cache, time.Hour)

	cache.On("Set", ctx, "key1", "value", time.Hour).Return(resp)
	resp.On("Err").Return(nil)

	err := repo.Set(ctx, "key1", "value")
	assert.NoError(t, err)

	cache.AssertExpectations(t)
	resp.AssertExpectations(t)
}

func TestCachedDocsRepo_Set_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := new(mockCache)
	resp := new(mockResponse[string])
	repo := New(cache, time.Hour)

	cache.On("Set", ctx, "key1", "value", time.Hour).Return(resp)
	resp.On("Err").Return(errors.New("set error"))

	err := repo.Set(ctx, "key1", "value")
	assert.Error(t, err)
	assert.EqualError(t, err, "set error")

	cache.AssertExpectations(t)
	resp.AssertExpectations(t)
}

func TestCachedDocsRepo_Get_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := new(mockCache)
	resp := new(mockResponse[string])
	repo := New(cache, time.Hour)

	cache.On("Get", ctx, "key1").Return(resp)
	resp.On("Result").Return("doc-json", nil)

	result, err := repo.Get(ctx, "key1")
	assert.NoError(t, err)
	assert.Equal(t, "doc-json", result)

	cache.AssertExpectations(t)
	resp.AssertExpectations(t)
}

func TestCachedDocsRepo_Get_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := new(mockCache)
	resp := new(mockResponse[string])
	repo := New(cache, time.Hour)

	cache.On("Get", ctx, "key1").Return(resp)
	resp.On("Result").Return("", errors.New("get error"))

	result, err := repo.Get(ctx, "key1")
	assert.Error(t, err)
	assert.EqualError(t, err, "get error")
	assert.Equal(t, "", result)

	cache.AssertExpectations(t)
	resp.AssertExpectations(t)
}

func TestCachedDocsRepo_Del_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := new(mockCache)
	resp := new(mockResponse[int64])
	repo := New(cache, time.Hour)

	cache.On("Del", ctx, []string{"key1", "key2"}).Return(resp)
	resp.On("Err").Return(nil)

	err := repo.Del(ctx, "key1", "key2")
	assert.NoError(t, err)

	cache.AssertExpectations(t)
	resp.AssertExpectations(t)
}

func TestCachedDocsRepo_Del_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cache := new(mockCache)
	resp := new(mockResponse[int64])
	repo := New(cache, time.Hour)

	cache.On("Del", ctx, []string{"key1", "key2"}).Return(resp)
	resp.On("Err").Return(errors.New("del error"))

	err := repo.Del(ctx, "key1", "key2")
	assert.Error(t, err)
	assert.EqualError(t, err, "del error")

	cache.AssertExpectations(t)
	resp.AssertExpectations(t)
}
