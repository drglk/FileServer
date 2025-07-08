package cachesessionrepo

import (
	"context"
	"fileserver/internal/models"
	cacherepo "fileserver/internal/repositories/cache"
	"time"
)

type repository struct {
	cache      cacherepo.Cache
	sessionTTL time.Duration
}

func New(cache cacherepo.Cache, sessionTTL time.Duration) *repository {
	return &repository{
		cache:      cache,
		sessionTTL: sessionTTL,
	}
}

func (r *repository) SaveSession(ctx context.Context, token string, userJSON string) error {
	return r.cache.Set(ctx, token, userJSON, r.sessionTTL).Err()
}

func (r *repository) DeleteSession(ctx context.Context, token string) error {
	return r.cache.Del(ctx, token).Err()
}

func (r *repository) GetUserByToken(ctx context.Context, token string) (string, error) {
	userJSON, err := r.cache.Get(ctx, token).Result()
	if err != nil {
		return "", err
	}

	if userJSON == "" {
		return "", models.ErrSessionNotFound
	}

	return userJSON, nil
}
