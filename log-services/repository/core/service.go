package core

import (
	"context"
	"log/slog"
	"strings"
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

func (s *Service) CreateLog(ctx context.Context, filePath string) (int64, error) {
	if strings.TrimSpace(filePath) == "" {
		return 0, ErrBadArguments
	}

	return s.db.CreateLog(ctx, filePath)
}

func (s *Service) SaveParsedLog(ctx context.Context, logID int64, parsed ParsedLog) (SaveParsedLogResult, error) {
	if logID <= 0 {
		return SaveParsedLogResult{}, ErrBadArguments
	}

	return s.db.SaveParsedLog(ctx, logID, parsed)
}

func (s *Service) FailLog(ctx context.Context, logID int64, errorText string) error {
	if logID <= 0 {
		return ErrBadArguments
	}

	return s.db.FailLog(ctx, logID, errorText)
}

func (s *Service) GetLog(ctx context.Context, logID int64) (Log, error) {
	if logID <= 0 {
		return Log{}, ErrBadArguments
	}

	return s.db.GetLog(ctx, logID)
}

func (s *Service) GetNode(ctx context.Context, nodeID int64) (Node, error) {
	if nodeID <= 0 {
		return Node{}, ErrBadArguments
	}

	return s.db.GetNode(ctx, nodeID)
}

func (s *Service) GetPortsByNode(ctx context.Context, nodeID int64) ([]Port, error) {
	if nodeID <= 0 {
		return nil, ErrBadArguments
	}

	return s.db.GetPortsByNode(ctx, nodeID)
}

func (s *Service) GetNodesByLog(ctx context.Context, logID int64) ([]Node, error) {
	if logID <= 0 {
		return nil, ErrBadArguments
	}

	return s.db.GetNodesByLog(ctx, logID)
}

func (s *Service) GetPortsByLog(ctx context.Context, logID int64) ([]Port, error) {
	if logID <= 0 {
		return nil, ErrBadArguments
	}

	return s.db.GetPortsByLog(ctx, logID)
}

func (s *Service) GetTopologyData(ctx context.Context, logID int64) (TopologyData, error) {
	if logID <= 0 {
		return TopologyData{}, ErrBadArguments
	}

	return s.db.GetTopologyData(ctx, logID)
}
