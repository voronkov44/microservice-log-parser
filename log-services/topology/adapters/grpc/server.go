package grpc

import (
	"context"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	topologypb "github.com/voronkov44/microservice-log-parser/log-services/proto/topology"
	"github.com/voronkov44/microservice-log-parser/log-services/topology/core"
)

type Server struct {
	topologypb.UnimplementedTopologyServer
	log     *slog.Logger
	service core.Topology
}

func NewServer(log *slog.Logger, service core.Topology) *Server {
	return &Server{
		log:     log,
		service: service,
	}
}

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Ping(ctx); err != nil {
		s.log.Warn("topology ping failed", "error", err)
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	return &emptypb.Empty{}, nil
}
