package db

import (
	"context"
	"errors"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	instance *Queries
	pool     *pgxpool.Pool
	once     sync.Once
	openErr  error
)

func Open(ctx context.Context, dsn string) (*Queries, *pgxpool.Pool, error) {
	// The API process supports one DB pool; the first DSN passed here wins.
	once.Do(func() {
		if dsn == "" {
			openErr = errors.New("DATABASE_URL is required")
			return
		}

		config, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			openErr = err
			return
		}
		config.MaxConns = 5

		p, err := pgxpool.NewWithConfig(ctx, config)
		if err != nil {
			openErr = err
			return
		}

		pool = p
		instance = New(pool)
	})

	return instance, pool, openErr
}
