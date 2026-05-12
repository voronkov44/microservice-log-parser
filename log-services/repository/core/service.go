package core

import (
	"context"
	"log/slog"
)

type Service struct {
	log *slog.Logger
	db  DB
}

func NewService(log *slog.Logger, db DB) *Service {
	return &Service{
		log: log,
		db:  db,
	}
}

func (s *Service) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}
