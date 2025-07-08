package cachedocsrepo

import (
	"context"
	cacherepo "fileserver/internal/repositories/cache"
	"time"
)

type repository struct {
	cache       cacherepo.Cache
	documentTTL time.Duration
}

func New(cache cacherepo.Cache, documentTTL time.Duration) *repository {
	return &repository{
		cache:       cache,
		documentTTL: documentTTL,
	}
}

func (r *repository) Get(ctx context.Context, key string) (string, error) {
	docJSON, err := r.cache.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	return docJSON, nil
}

func (r *repository) Set(ctx context.Context, key string, value interface{}) error {
	return r.cache.Set(ctx, key, value, r.documentTTL).Err()
}

func (r *repository) Del(ctx context.Context, keys ...string) error {
	return r.cache.Del(ctx, keys...).Err()
}
