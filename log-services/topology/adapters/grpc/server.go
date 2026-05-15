package grpc

import (
	"context"
	"errors"
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
		return nil, grpcError(err)
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) GetTopology(ctx context.Context, req *topologypb.GetTopologyRequest) (*topologypb.TopologyResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	topology, err := s.service.GetTopology(ctx, req.GetLogId())
	if err != nil {
		s.log.Warn("get topology failed",
			"log_id", req.GetLogId(),
			"error", err,
		)

		return nil, grpcError(err)
	}

	return topologyToProto(topology), nil
}

func topologyToProto(in core.TopologyResult) *topologypb.TopologyResponse {
	return &topologypb.TopologyResponse{
		LogId:   in.LogID,
		Summary: topologySummaryToProto(in.Summary),
		Nodes:   topologyNodesToProto(in.Nodes),
		Groups:  topologyGroupsToProto(in.Groups),
		Edges:   topologyEdgesToProto(in.Edges),
	}
}

func topologySummaryToProto(in core.TopologySummary) *topologypb.TopologySummary {
	return &topologypb.TopologySummary{
		NodesCount:    in.NodesCount,
		PortsCount:    in.PortsCount,
		EdgesCount:    in.EdgesCount,
		HostsCount:    in.HostsCount,
		SwitchesCount: in.SwitchesCount,
	}
}

func topologyNodesToProto(in []core.TopologyNode) []*topologypb.TopologyNode {
	out := make([]*topologypb.TopologyNode, 0, len(in))

	for _, node := range in {
		out = append(out, &topologypb.TopologyNode{
			Id:                 node.ID,
			LogId:              node.LogID,
			NodeGuid:           node.NodeGUID,
			NodeDesc:           node.NodeDesc,
			NodeType:           node.NodeType,
			NodeKind:           node.NodeKind,
			DeclaredPortsCount: node.DeclaredPortsCount,
			ParsedPortsCount:   node.ParsedPortsCount,
			SerialNumber:       node.SerialNumber,
			ProductName:        node.ProductName,
		})
	}

	return out
}

func topologyGroupsToProto(in []core.TopologyGroup) []*topologypb.TopologyGroup {
	out := make([]*topologypb.TopologyGroup, 0, len(in))

	for _, group := range in {
		out = append(out, &topologypb.TopologyGroup{
			Name:      group.Name,
			Kind:      group.Kind,
			NodeIds:   group.NodeIDs,
			NodeGuids: group.NodeGUIDs,
		})
	}

	return out
}

func topologyEdgesToProto(in []core.TopologyEdge) []*topologypb.TopologyEdge {
	out := make([]*topologypb.TopologyEdge, 0, len(in))

	for _, edge := range in {
		out = append(out, &topologypb.TopologyEdge{
			SourceNodeId:    edge.SourceNodeID,
			SourceNodeGuid:  edge.SourceNodeGUID,
			SourcePortNum:   edge.SourcePortNum,
			SourcePortGuid:  edge.SourcePortGUID,
			TargetNodeId:    edge.TargetNodeID,
			TargetNodeGuid:  edge.TargetNodeGUID,
			TargetPortNum:   edge.TargetPortNum,
			TargetPortGuid:  edge.TargetPortGUID,
			Relation:        edge.Relation,
			LinkWidthActive: edge.LinkWidthActive,
			LinkSpeedActive: edge.LinkSpeedActive,
			PortState:       edge.PortState,
		})
	}

	return out
}

func grpcError(err error) error {
	switch {
	case errors.Is(err, core.ErrBadArguments):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, core.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, core.ErrLogNotParsed):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, core.ErrUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
