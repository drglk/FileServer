package postgres

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	_ "github.com/lib/pq"
)

const pkg = "postgres"

func New(ctx context.Context, cfg Config) (*sqlx.DB, error) {
	op := pkg + "New"

	dataSource := fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=disable", cfg.User, cfg.Password, cfg.Addr, cfg.Port, cfg.DB,
	)

	conn, err := sqlx.ConnectContext(ctx, "postgres", dataSource)
	if err != nil {
		return nil, fmt.Errorf("%s: sqlx connect: %w", op, err)
	}

	err = conn.PingContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: ping failed: %w", op, err)
	}

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	return conn, nil
}
