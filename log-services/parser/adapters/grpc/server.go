package grpc

import (
	"context"
	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
	"github.com/voronkov44/microservice-log-parser/log-services/proto/parser"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"log/slog"
)

type Server struct {
	parser.UnimplementedParserServer
	log     *slog.Logger
	service core.Parser
}

func NewServer(log *slog.Logger, service core.Parser) *Server {
	return &Server{
		log:     log,
		service: service,
	}
}

func (s *Server) Ping(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.service.Ping(ctx); err != nil {
		s.log.Warn("parser ping failed", "error", err)
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	return &emptypb.Empty{}, nil
}
