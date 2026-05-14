package core

import (
	"context"
	"log/slog"
	"strings"
)

type Service struct {
	log        *slog.Logger
	repository Repository
	parser     Parser
}

func NewService(log *slog.Logger, repository Repository, parser Parser) *Service {
	return &Service{
		log:        log,
		repository: repository,
		parser:     parser,
	}
}

func (s *Service) ParseLog(ctx context.Context, path string) (ParseLogResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return ParseLogResult{}, ErrBadArguments
	}

	logID, err := s.repository.CreateLog(ctx, path)
	if err != nil {
		return ParseLogResult{}, err
	}

	parsed, err := s.parser.Parse(ctx, path)
	if err != nil {
		s.failLog(ctx, logID, err)
		return ParseLogResult{}, err
	}

	saveResult, err := s.repository.SaveParsedLog(ctx, logID, parsed)
	if err != nil {
		s.failLog(ctx, logID, err)
		return ParseLogResult{}, err
	}

	return ParseLogResult{
		LogID:          saveResult.LogID,
		Status:         LogStatusParsed,
		NodesCount:     saveResult.NodesCount,
		PortsCount:     saveResult.PortsCount,
		NodesInfoCount: int32(len(parsed.NodesInfo)),
	}, nil
}

func (s *Service) failLog(ctx context.Context, logID int64, cause error) {
	if logID <= 0 || cause == nil {
		return
	}

	if err := s.repository.FailLog(ctx, logID, cause.Error()); err != nil {
		s.log.Warn("failed to mark log as failed",
			"log_id", logID,
			"original_error", cause,
			"fail_error", err,
		)
	}
}
