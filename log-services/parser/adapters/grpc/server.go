package grpc

import (
	"context"
	"errors"
	"log/slog"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/voronkov44/microservice-log-parser/log-services/parser/core"
	parserpb "github.com/voronkov44/microservice-log-parser/log-services/proto/parser"
)

type Server struct {
	parserpb.UnimplementedParserServer
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
		return nil, grpcError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) Parse(ctx context.Context, req *parserpb.ParseRequest) (*parserpb.ParseResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	parsed, err := s.service.Parse(ctx, req.GetPath())
	if err != nil {
		s.log.Warn("parse failed", "path", req.GetPath(), "error", err)
		return nil, grpcError(err)
	}

	return &parserpb.ParseResponse{
		Log: parsedLogToProto(parsed),
	}, nil
}

func parsedLogToProto(in core.ParsedLog) *parserpb.ParsedLog {
	nodes := make([]*parserpb.Node, 0, len(in.Nodes))
	for _, node := range in.Nodes {
		nodes = append(nodes, &parserpb.Node{
			NodeGuid:        node.NodeGUID,
			NodeDesc:        node.NodeDesc,
			NodeType:        node.NodeType,
			NodeKind:        node.NodeKind,
			NumPorts:        node.NumPorts,
			ClassVersion:    node.ClassVersion,
			BaseVersion:     node.BaseVersion,
			SystemImageGuid: node.SystemImageGUID,
			PortGuid:        node.PortGUID,
			RawJson:         node.RawJSON,
		})
	}

	ports := make([]*parserpb.Port, 0, len(in.Ports))
	for _, port := range in.Ports {
		ports = append(ports, &parserpb.Port{
			NodeGuid:        port.NodeGUID,
			PortGuid:        port.PortGUID,
			PortNum:         port.PortNum,
			Lid:             port.LID,
			LocalPortNum:    port.LocalPortNum,
			PortState:       port.PortState,
			PortPhyState:    port.PortPhyState,
			LinkWidthActive: port.LinkWidthActive,
			LinkSpeedActive: port.LinkSpeedActive,
			RawJson:         port.RawJSON,
		})
	}

	nodesInfo := make([]*parserpb.NodeInfo, 0, len(in.NodesInfo))
	for _, info := range in.NodesInfo {
		nodesInfo = append(nodesInfo, &parserpb.NodeInfo{
			NodeGuid:     info.NodeGUID,
			SerialNumber: info.SerialNumber,
			PartNumber:   info.PartNumber,
			Revision:     info.Revision,
			ProductName:  info.ProductName,
			RawJson:      info.RawJSON,
		})
	}

	return &parserpb.ParsedLog{
		Nodes:     nodes,
		Ports:     ports,
		NodesInfo: nodesInfo,
	}
}

func grpcError(err error) error {
	switch {
	case errors.Is(err, core.ErrBadArguments):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, core.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, core.ErrUnsupportedFormat):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, core.ErrParse):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
