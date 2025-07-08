package redis

import (
	"context"
	"errors"
	cacherepo "fileserver/internal/repositories/cache"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const pkg = "redis/"

type Client struct {
	redisClient *redis.Client
}

type redisResponse[T any] struct {
	cmd redis.Cmder
	get func() (T, error)
}

func (r redisResponse[T]) Err() error {
	err := r.cmd.Err()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	return err
}

func (r redisResponse[T]) Result() (T, error) {
	res, err := r.get()
	if errors.Is(err, redis.Nil) {
		var zero T
		return zero, nil
	}

	return res, err
}

func (c *Client) Get(ctx context.Context, key string) cacherepo.CacheResponse[string] {
	cmd := c.redisClient.Get(ctx, key)
	return redisResponse[string]{
		cmd: cmd,
		get: cmd.Result,
	}
}

func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) cacherepo.CacheResponse[string] {
	cmd := c.redisClient.Set(ctx, key, value, expiration)
	return redisResponse[string]{
		cmd: cmd,
		get: cmd.Result,
	}
}

func (c *Client) Del(ctx context.Context, keys ...string) cacherepo.CacheResponse[int64] {
	cmd := c.redisClient.Del(ctx, keys...)
	return redisResponse[int64]{
		cmd: cmd,
		get: cmd.Result,
	}
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	op := pkg + "New"

	client := &Client{
		redisClient: redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}),
	}

	if err := client.redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: redis: ping failed: %w", op, err)
	}

	return client, nil
}
