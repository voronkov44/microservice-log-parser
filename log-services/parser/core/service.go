package core

import (
	"context"
	"log/slog"
	"strings"
)

type Service struct {
	log    *slog.Logger
	engine Engine
}

func NewService(log *slog.Logger, engine Engine) *Service {
	return &Service{
		log:    log,
		engine: engine,
	}
}

func (s *Service) Ping(_ context.Context) error {
	return nil
}

func (s *Service) Parse(ctx context.Context, path string) (ParsedLog, error) {
	if strings.TrimSpace(path) == "" {
		return ParsedLog{}, ErrBadArguments
	}

	return s.engine.Parse(ctx, path)
}
