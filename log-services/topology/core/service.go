package core

import (
	"context"
	"log/slog"
)

type Service struct {
	log               *slog.Logger
	repositoryAddress string
}

func NewService(log *slog.Logger, repositoryAddress string) *Service {
	return &Service{
		log:               log,
		repositoryAddress: repositoryAddress,
	}
}

func (s *Service) Ping(_ context.Context) error {
	return nil
}
