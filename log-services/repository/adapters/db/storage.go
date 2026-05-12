package db

import (
	"context"
	"fmt"
	"log/slog"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	log  *slog.Logger
	conn *sqlx.DB
}

func New(log *slog.Logger, address string) (*DB, error) {
	conn, err := sqlx.Connect("pgx", address)
	if err != nil {
		log.Error("connection problem", "address", address, "error", err)
		return nil, fmt.Errorf("connect db: %w", err)
	}

	return &DB{
		log:  log,
		conn: conn,
	}, nil
}

func (db *DB) Ping(ctx context.Context) error {
	if err := db.conn.PingContext(ctx); err != nil {
		db.log.Warn("db ping failed", "error", err)
		return fmt.Errorf("ping db: %w", err)
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
