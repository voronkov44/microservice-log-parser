package core

import (
	"context"
	"log/slog"
)

type Service struct {
	log *slog.Logger
}

func NewService(log *slog.Logger) *Service {
	return &Service{
		log: log,
	}
}

func (s *Service) Ping(_ context.Context) error {
	return nil
}
