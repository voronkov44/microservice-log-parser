package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

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
		return nil, grpcError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) CreateLog(ctx context.Context, req *repositorypb.CreateLogRequest) (*repositorypb.CreateLogResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	logID, err := s.service.CreateLog(ctx, req.GetFilePath())
	if err != nil {
		return nil, grpcError(err)
	}

	return &repositorypb.CreateLogResponse{
		LogId: logID,
	}, nil
}

func (s *Server) SaveParsedLog(ctx context.Context, req *repositorypb.SaveParsedLogRequest) (*repositorypb.SaveParsedLogResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	parsed, err := parsedLogFromProto(req.GetParsedLog())
	if err != nil {
		return nil, grpcError(err)
	}

	result, err := s.service.SaveParsedLog(ctx, req.GetLogId(), parsed)
	if err != nil {
		return nil, grpcError(err)
	}

	return &repositorypb.SaveParsedLogResponse{
		LogId:      result.LogID,
		NodesCount: result.NodesCount,
		PortsCount: result.PortsCount,
	}, nil
}

func (s *Server) FailLog(ctx context.Context, req *repositorypb.FailLogRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	if err := s.service.FailLog(ctx, req.GetLogId(), req.GetError()); err != nil {
		return nil, grpcError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetLog(ctx context.Context, req *repositorypb.GetLogRequest) (*repositorypb.Log, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	log, err := s.service.GetLog(ctx, req.GetLogId())
	if err != nil {
		return nil, grpcError(err)
	}

	return logToProto(log), nil
}

func (s *Server) GetNode(ctx context.Context, req *repositorypb.GetNodeRequest) (*repositorypb.NodeDetails, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	node, err := s.service.GetNode(ctx, req.GetNodeId())
	if err != nil {
		return nil, grpcError(err)
	}

	return nodeToProto(node), nil
}

func (s *Server) GetPortsByNode(ctx context.Context, req *repositorypb.GetPortsByNodeRequest) (*repositorypb.PortsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	ports, err := s.service.GetPortsByNode(ctx, req.GetNodeId())
	if err != nil {
		return nil, grpcError(err)
	}

	return &repositorypb.PortsResponse{
		Ports: portsToProto(ports),
	}, nil
}

func (s *Server) GetNodesByLog(ctx context.Context, req *repositorypb.GetNodesByLogRequest) (*repositorypb.NodesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	nodes, err := s.service.GetNodesByLog(ctx, req.GetLogId())
	if err != nil {
		return nil, grpcError(err)
	}

	return &repositorypb.NodesResponse{
		Nodes: nodesToProto(nodes),
	}, nil
}

func (s *Server) GetPortsByLog(ctx context.Context, req *repositorypb.GetPortsByLogRequest) (*repositorypb.PortsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	ports, err := s.service.GetPortsByLog(ctx, req.GetLogId())
	if err != nil {
		return nil, grpcError(err)
	}

	return &repositorypb.PortsResponse{
		Ports: portsToProto(ports),
	}, nil
}

func (s *Server) GetTopologyData(ctx context.Context, req *repositorypb.GetTopologyDataRequest) (*repositorypb.TopologyDataResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	data, err := s.service.GetTopologyData(ctx, req.GetLogId())
	if err != nil {
		return nil, grpcError(err)
	}

	return &repositorypb.TopologyDataResponse{
		Log:   logToProto(data.Log),
		Nodes: nodesToProto(data.Nodes),
		Ports: portsToProto(data.Ports),
	}, nil
}

func parsedLogFromProto(in *repositorypb.ParsedLog) (core.ParsedLog, error) {
	if in == nil {
		return core.ParsedLog{}, core.ErrBadArguments
	}

	nodes := make([]core.Node, 0, len(in.GetNodes()))
	for _, node := range in.GetNodes() {
		if node == nil {
			continue
		}

		nodes = append(nodes, core.Node{
			NodeGUID:        node.GetNodeGuid(),
			NodeDesc:        node.GetNodeDesc(),
			NodeType:        node.GetNodeType(),
			NodeKind:        node.GetNodeKind(),
			NumPorts:        node.GetNumPorts(),
			ClassVersion:    node.GetClassVersion(),
			BaseVersion:     node.GetBaseVersion(),
			SystemImageGUID: node.GetSystemImageGuid(),
			PortGUID:        node.GetPortGuid(),
			RawJSON:         node.GetRawJson(),
		})
	}

	ports := make([]core.Port, 0, len(in.GetPorts()))
	for _, port := range in.GetPorts() {
		if port == nil {
			continue
		}

		ports = append(ports, core.Port{
			NodeGUID:        port.GetNodeGuid(),
			PortGUID:        port.GetPortGuid(),
			PortNum:         port.GetPortNum(),
			LID:             port.GetLid(),
			LocalPortNum:    port.GetLocalPortNum(),
			PortState:       port.GetPortState(),
			PortPhyState:    port.GetPortPhyState(),
			LinkWidthActive: port.GetLinkWidthActive(),
			LinkSpeedActive: port.GetLinkSpeedActive(),
			RawJSON:         port.GetRawJson(),
		})
	}

	nodesInfo := make([]core.NodeInfo, 0, len(in.GetNodesInfo()))
	for _, info := range in.GetNodesInfo() {
		if info == nil {
			continue
		}

		nodesInfo = append(nodesInfo, core.NodeInfo{
			NodeGUID:     info.GetNodeGuid(),
			SerialNumber: info.GetSerialNumber(),
			PartNumber:   info.GetPartNumber(),
			Revision:     info.GetRevision(),
			ProductName:  info.GetProductName(),
			RawJSON:      info.GetRawJson(),
		})
	}

	return core.ParsedLog{
		Nodes:     nodes,
		Ports:     ports,
		NodesInfo: nodesInfo,
	}, nil
}

func logToProto(in core.Log) *repositorypb.Log {
	return &repositorypb.Log{
		Id:         in.ID,
		FilePath:   in.FilePath,
		Status:     logStatusToProto(in.Status),
		NodesCount: in.NodesCount,
		PortsCount: in.PortsCount,
		Error:      in.Error,
		UploadedAt: timestampToProto(in.UploadedAt),
		ParsedAt:   timestampToProto(in.ParsedAt),
	}
}

func timestampToProto(value string) *timestamppb.Timestamp {
	if value == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}

	return timestamppb.New(t)
}

func logStatusToProto(status core.LogStatus) repositorypb.LogStatus {
	switch status {
	case core.LogStatusProcessing:
		return repositorypb.LogStatus_LOG_STATUS_PROCESSING
	case core.LogStatusParsed:
		return repositorypb.LogStatus_LOG_STATUS_PARSED
	case core.LogStatusFailed:
		return repositorypb.LogStatus_LOG_STATUS_FAILED
	default:
		return repositorypb.LogStatus_LOG_STATUS_UNSPECIFIED
	}
}

func nodeToProto(in core.Node) *repositorypb.NodeDetails {
	return &repositorypb.NodeDetails{
		Id:              in.ID,
		LogId:           in.LogID,
		NodeGuid:        in.NodeGUID,
		NodeDesc:        in.NodeDesc,
		NodeType:        in.NodeType,
		NodeKind:        in.NodeKind,
		NumPorts:        in.NumPorts,
		ClassVersion:    in.ClassVersion,
		BaseVersion:     in.BaseVersion,
		SystemImageGuid: in.SystemImageGUID,
		PortGuid:        in.PortGUID,
		Info:            nodeInfoToProto(in.Info),
		RawJson:         in.RawJSON,
	}
}

func nodesToProto(in []core.Node) []*repositorypb.NodeDetails {
	out := make([]*repositorypb.NodeDetails, 0, len(in))
	for _, node := range in {
		out = append(out, nodeToProto(node))
	}

	return out
}

func nodeInfoToProto(in *core.NodeInfo) *repositorypb.NodeInfo {
	if in == nil {
		return nil
	}

	return &repositorypb.NodeInfo{
		Id:           in.ID,
		NodeId:       in.NodeID,
		NodeGuid:     in.NodeGUID,
		SerialNumber: in.SerialNumber,
		PartNumber:   in.PartNumber,
		Revision:     in.Revision,
		ProductName:  in.ProductName,
		RawJson:      in.RawJSON,
	}
}

func portToProto(in core.Port) *repositorypb.Port {
	return &repositorypb.Port{
		Id:              in.ID,
		LogId:           in.LogID,
		NodeId:          in.NodeID,
		NodeGuid:        in.NodeGUID,
		PortGuid:        in.PortGUID,
		PortNum:         in.PortNum,
		Lid:             in.LID,
		LocalPortNum:    in.LocalPortNum,
		PortState:       in.PortState,
		PortPhyState:    in.PortPhyState,
		LinkWidthActive: in.LinkWidthActive,
		LinkSpeedActive: in.LinkSpeedActive,
		RawJson:         in.RawJSON,
	}
}

func portsToProto(in []core.Port) []*repositorypb.Port {
	out := make([]*repositorypb.Port, 0, len(in))
	for _, port := range in {
		out = append(out, portToProto(port))
	}

	return out
}

func grpcError(err error) error {
	switch {
	case errors.Is(err, core.ErrBadArguments):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, core.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, core.ErrInvalidStatus):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, core.ErrUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
