package cacherepo

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) CacheResponse[string]
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) CacheResponse[string]
	Del(ctx context.Context, keys ...string) CacheResponse[int64]
}

type CacheResponse[T any] interface {
	Err() error
	Result() (T, error)
}
