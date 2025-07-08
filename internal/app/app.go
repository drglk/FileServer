package app

import (
	"context"
	"fileserver/internal/cache/redis"
	"fileserver/internal/config"
	"fileserver/internal/dbs/postgres"
	cachedocsrepo "fileserver/internal/repositories/cache/docs"
	cachesessionrepo "fileserver/internal/repositories/cache/session"
	documentrepo "fileserver/internal/repositories/db/document"
	userrepo "fileserver/internal/repositories/db/user"
	filerepo "fileserver/internal/repositories/storage/file"
	authservice "fileserver/internal/services/auth"
	documentservice "fileserver/internal/services/document"
	userservice "fileserver/internal/services/user"
	"fmt"
	"log/slog"
)

type App struct {
	AuthService     AuthService
	UserService     UserService
	DocumentService DocumentService
}

func NewApp(ctx context.Context, log *slog.Logger, dbCfg config.DB, cacheConfig config.Cache, fileStorageCfg config.FileStorage, adminToken string) (*App, error) {
	db, err := postgres.New(ctx, postgres.Config{
		Addr:     dbCfg.Addr,
		Port:     dbCfg.Port,
		User:     dbCfg.User,
		Password: dbCfg.Password,
		DB:       dbCfg.DB})
	if err != nil {
		log.Error("failed connect to db", "err", err)
		return nil, fmt.Errorf("failed connect to db: %w", err)
	}

	cache, err := redis.New(ctx, redis.Config{Addr: cacheConfig.Addr, Password: cacheConfig.Password, DB: cacheConfig.DB})
	if err != nil {
		log.Error("failed connect to cache", "err", err)
		return nil, fmt.Errorf("failed connect to cache: %w", err)
	}

	userRepo := userrepo.NewRepository(db)

	sessionCacheRepo := cachesessionrepo.New(cache, cacheConfig.SessionTTL)

	documentCacheRepo := cachedocsrepo.New(cache, cacheConfig.DocumentsTTL)

	userService := userservice.New(log, userRepo, userRepo)

	authService := authservice.New(log, userService, userService, sessionCacheRepo, adminToken)

	docRepo := documentrepo.NewRepository(db)

	fileStorage := filerepo.NewRepository(fileStorageCfg.Path)

	documentService := documentservice.New(log, docRepo, documentCacheRepo, fileStorage)

	return &App{
		AuthService:     authService,
		UserService:     userService,
		DocumentService: documentService,
	}, nil
}
