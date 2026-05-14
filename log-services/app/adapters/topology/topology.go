package topology

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	topologypb "github.com/voronkov44/microservice-log-parser/log-services/proto/topology"
)

type Client struct {
	log    *slog.Logger
	client topologypb.TopologyClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("new grpc client for topology %s: %w", address, err)
	}

	return &Client{
		log:    log,
		client: topologypb.NewTopologyClient(conn),
		conn:   conn,
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.client.Ping(ctx, &emptypb.Empty{})
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return core.ErrUnavailable
		default:
			return err
		}
	}

	return nil
}

func (c *Client) GetTopology(ctx context.Context, logID int64) (core.Topology, error) {
	resp, err := c.client.GetTopology(ctx, &topologypb.GetTopologyRequest{
		LogId: logID,
	})
	if err != nil {
		return core.Topology{}, mapGRPCError(err)
	}

	return topologyFromProto(resp), nil
}

func topologyFromProto(in *topologypb.TopologyResponse) core.Topology {
	if in == nil {
		return core.Topology{}
	}

	return core.Topology{
		LogID:   in.GetLogId(),
		Summary: topologySummaryFromProto(in.GetSummary()),
		Nodes:   topologyNodesFromProto(in.GetNodes()),
		Groups:  topologyGroupsFromProto(in.GetGroups()),
		Edges:   topologyEdgesFromProto(in.GetEdges()),
	}
}

func topologySummaryFromProto(in *topologypb.TopologySummary) core.TopologySummary {
	if in == nil {
		return core.TopologySummary{}
	}

	return core.TopologySummary{
		NodesCount:    in.GetNodesCount(),
		PortsCount:    in.GetPortsCount(),
		EdgesCount:    in.GetEdgesCount(),
		HostsCount:    in.GetHostsCount(),
		SwitchesCount: in.GetSwitchesCount(),
	}
}

func topologyNodesFromProto(in []*topologypb.TopologyNode) []core.TopologyNode {
	out := make([]core.TopologyNode, 0, len(in))

	for _, node := range in {
		if node == nil {
			continue
		}

		out = append(out, core.TopologyNode{
			ID:                 node.GetId(),
			LogID:              node.GetLogId(),
			NodeGUID:           node.GetNodeGuid(),
			NodeDesc:           node.GetNodeDesc(),
			NodeType:           node.GetNodeType(),
			NodeKind:           node.GetNodeKind(),
			DeclaredPortsCount: node.GetDeclaredPortsCount(),
			ParsedPortsCount:   node.GetParsedPortsCount(),
			SerialNumber:       node.GetSerialNumber(),
			ProductName:        node.GetProductName(),
		})
	}

	return out
}

func topologyGroupsFromProto(in []*topologypb.TopologyGroup) []core.TopologyGroup {
	out := make([]core.TopologyGroup, 0, len(in))

	for _, group := range in {
		if group == nil {
			continue
		}

		out = append(out, core.TopologyGroup{
			Name:      group.GetName(),
			Kind:      group.GetKind(),
			NodeIDs:   group.GetNodeIds(),
			NodeGUIDs: group.GetNodeGuids(),
		})
	}

	return out
}

func topologyEdgesFromProto(in []*topologypb.TopologyEdge) []core.TopologyEdge {
	out := make([]core.TopologyEdge, 0, len(in))

	for _, edge := range in {
		if edge == nil {
			continue
		}

		out = append(out, core.TopologyEdge{
			SourceNodeID:    edge.GetSourceNodeId(),
			SourceNodeGUID:  edge.GetSourceNodeGuid(),
			SourcePortNum:   edge.GetSourcePortNum(),
			SourcePortGUID:  edge.GetSourcePortGuid(),
			TargetNodeID:    edge.GetTargetNodeId(),
			TargetNodeGUID:  edge.GetTargetNodeGuid(),
			TargetPortNum:   edge.GetTargetPortNum(),
			TargetPortGUID:  edge.GetTargetPortGuid(),
			Relation:        edge.GetRelation(),
			LinkWidthActive: edge.GetLinkWidthActive(),
			LinkSpeedActive: edge.GetLinkSpeedActive(),
			PortState:       edge.GetPortState(),
		})
	}

	return out
}

func mapGRPCError(err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return core.ErrBadArguments
	case codes.NotFound:
		return core.ErrNotFound
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
		return core.ErrUnavailable
	default:
		return err
	}
}

var _ core.TopologyProvider = (*Client)(nil)
