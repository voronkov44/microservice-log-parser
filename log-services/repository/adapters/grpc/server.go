package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	repositorypb "github.com/voronkov44/microservice-log-parser/log-services/proto/repository"
	"github.com/voronkov44/microservice-log-parser/log-services/repository/core"
)

type Server struct {
	repositorypb.UnimplementedRepositoryServer
	log     *slog.Logger
	service core.Repository
}

func NewServer(log *slog.Logger, service core.Repository) *Server {
	return &Server{
		log:     log,
		service: service,
	}
}

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Ping(ctx); err != nil {
		s.log.Warn("repository ping failed", "error", err)
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	return &emptypb.Empty{}, nil
}
