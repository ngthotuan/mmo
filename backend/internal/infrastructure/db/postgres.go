package db

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"mmo/pkg/config"
	"mmo/pkg/logger"
)

func New(cfg config.DBConfig) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.PingTimeout)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	logger.Info("connected to postgres")
	return db, nil
}
