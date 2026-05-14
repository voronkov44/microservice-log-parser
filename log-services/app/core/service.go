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
	topology   TopologyProvider
}

func NewService(log *slog.Logger, repository Repository, parser Parser, topology TopologyProvider) *Service {
	return &Service{
		log:        log,
		repository: repository,
		parser:     parser,
		topology:   topology,
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

func (s *Service) GetLog(ctx context.Context, logID int64) (Log, error) {
	if logID <= 0 {
		return Log{}, ErrBadArguments
	}

	return s.repository.GetLog(ctx, logID)
}

func (s *Service) GetNode(ctx context.Context, nodeID int64) (Node, error) {
	if nodeID <= 0 {
		return Node{}, ErrBadArguments
	}

	return s.repository.GetNode(ctx, nodeID)
}

func (s *Service) GetNodesByLog(ctx context.Context, logID int64) ([]Node, error) {
	if logID <= 0 {
		return nil, ErrBadArguments
	}

	if _, err := s.repository.GetLog(ctx, logID); err != nil {
		return nil, err
	}

	return s.repository.GetNodesByLog(ctx, logID)
}

func (s *Service) GetPortsByLog(ctx context.Context, logID int64) ([]Port, error) {
	if logID <= 0 {
		return nil, ErrBadArguments
	}

	if _, err := s.repository.GetLog(ctx, logID); err != nil {
		return nil, err
	}

	return s.repository.GetPortsByLog(ctx, logID)
}

func (s *Service) GetPortsByNode(ctx context.Context, nodeID int64) ([]Port, error) {
	if nodeID <= 0 {
		return nil, ErrBadArguments
	}

	if _, err := s.repository.GetNode(ctx, nodeID); err != nil {
		return nil, err
	}

	return s.repository.GetPortsByNode(ctx, nodeID)
}

func (s *Service) GetTopology(ctx context.Context, logID int64) (Topology, error) {
	if logID <= 0 {
		return Topology{}, ErrBadArguments
	}

	return s.topology.GetTopology(ctx, logID)
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
