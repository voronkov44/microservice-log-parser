package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	repositorypb "github.com/voronkov44/microservice-log-parser/log-services/proto/repository"
)

type Client struct {
	log    *slog.Logger
	client repositorypb.RepositoryClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("new grpc client for repository %s: %w", address, err)
	}

	return &Client{
		log:    log,
		client: repositorypb.NewRepositoryClient(conn),
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

func (c *Client) CreateLog(ctx context.Context, filePath string) (int64, error) {
	resp, err := c.client.CreateLog(ctx, &repositorypb.CreateLogRequest{
		FilePath: filePath,
	})
	if err != nil {
		return 0, mapGRPCError(err)
	}

	return resp.GetLogId(), nil
}

func (c *Client) SaveParsedLog(ctx context.Context, logID int64, parsed core.ParsedLog) (core.SaveParsedLogResult, error) {
	resp, err := c.client.SaveParsedLog(ctx, &repositorypb.SaveParsedLogRequest{
		LogId:     logID,
		ParsedLog: parsedLogToProto(parsed),
	})
	if err != nil {
		return core.SaveParsedLogResult{}, mapGRPCError(err)
	}

	return core.SaveParsedLogResult{
		LogID:      resp.GetLogId(),
		NodesCount: resp.GetNodesCount(),
		PortsCount: resp.GetPortsCount(),
	}, nil
}

func (c *Client) FailLog(ctx context.Context, logID int64, errorText string) error {
	_, err := c.client.FailLog(ctx, &repositorypb.FailLogRequest{
		LogId: logID,
		Error: errorText,
	})
	if err != nil {
		return mapGRPCError(err)
	}

	return nil
}

func (c *Client) GetLog(ctx context.Context, logID int64) (core.Log, error) {
	resp, err := c.client.GetLog(ctx, &repositorypb.GetLogRequest{
		LogId: logID,
	})
	if err != nil {
		return core.Log{}, mapGRPCError(err)
	}

	return logFromProto(resp), nil
}

func (c *Client) GetNode(ctx context.Context, nodeID int64) (core.Node, error) {
	resp, err := c.client.GetNode(ctx, &repositorypb.GetNodeRequest{
		NodeId: nodeID,
	})
	if err != nil {
		return core.Node{}, mapGRPCError(err)
	}

	return nodeFromProto(resp), nil
}

func (c *Client) GetPortsByNode(ctx context.Context, nodeID int64) ([]core.Port, error) {
	resp, err := c.client.GetPortsByNode(ctx, &repositorypb.GetPortsByNodeRequest{
		NodeId: nodeID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return portsFromProto(resp.GetPorts()), nil
}

func (c *Client) GetNodesByLog(ctx context.Context, logID int64) ([]core.Node, error) {
	resp, err := c.client.GetNodesByLog(ctx, &repositorypb.GetNodesByLogRequest{
		LogId: logID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return nodesFromProto(resp.GetNodes()), nil
}

func (c *Client) GetPortsByLog(ctx context.Context, logID int64) ([]core.Port, error) {
	resp, err := c.client.GetPortsByLog(ctx, &repositorypb.GetPortsByLogRequest{
		LogId: logID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return portsFromProto(resp.GetPorts()), nil
}

func parsedLogToProto(in core.ParsedLog) *repositorypb.ParsedLog {
	nodes := make([]*repositorypb.ParsedNode, 0, len(in.Nodes))
	for _, node := range in.Nodes {
		nodes = append(nodes, &repositorypb.ParsedNode{
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

	ports := make([]*repositorypb.ParsedPort, 0, len(in.Ports))
	for _, port := range in.Ports {
		ports = append(ports, &repositorypb.ParsedPort{
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

	nodesInfo := make([]*repositorypb.ParsedNodeInfo, 0, len(in.NodesInfo))
	for _, info := range in.NodesInfo {
		nodesInfo = append(nodesInfo, &repositorypb.ParsedNodeInfo{
			NodeGuid:     info.NodeGUID,
			SerialNumber: info.SerialNumber,
			PartNumber:   info.PartNumber,
			Revision:     info.Revision,
			ProductName:  info.ProductName,
			RawJson:      info.RawJSON,
		})
	}

	return &repositorypb.ParsedLog{
		Nodes:     nodes,
		Ports:     ports,
		NodesInfo: nodesInfo,
	}
}

func logFromProto(in *repositorypb.Log) core.Log {
	if in == nil {
		return core.Log{}
	}

	return core.Log{
		ID:         in.GetId(),
		FilePath:   in.GetFilePath(),
		Status:     logStatusFromProto(in.GetStatus()),
		NodesCount: in.GetNodesCount(),
		PortsCount: in.GetPortsCount(),
		Error:      in.GetError(),
		UploadedAt: timestampFromProto(in.GetUploadedAt()),
		ParsedAt:   timestampFromProto(in.GetParsedAt()),
	}
}

func timestampFromProto(in *timestamppb.Timestamp) string {
	if in == nil {
		return ""
	}

	return in.AsTime().UTC().Format(time.RFC3339)
}

func logStatusFromProto(status repositorypb.LogStatus) core.LogStatus {
	switch status {
	case repositorypb.LogStatus_LOG_STATUS_PROCESSING:
		return core.LogStatusProcessing
	case repositorypb.LogStatus_LOG_STATUS_PARSED:
		return core.LogStatusParsed
	case repositorypb.LogStatus_LOG_STATUS_FAILED:
		return core.LogStatusFailed
	default:
		return ""
	}
}

func nodesFromProto(in []*repositorypb.NodeDetails) []core.Node {
	out := make([]core.Node, 0, len(in))

	for _, node := range in {
		if node == nil {
			continue
		}

		out = append(out, nodeFromProto(node))
	}

	return out
}

func nodeFromProto(in *repositorypb.NodeDetails) core.Node {
	if in == nil {
		return core.Node{}
	}

	return core.Node{
		ID:              in.GetId(),
		LogID:           in.GetLogId(),
		NodeGUID:        in.GetNodeGuid(),
		NodeDesc:        in.GetNodeDesc(),
		NodeType:        in.GetNodeType(),
		NodeKind:        in.GetNodeKind(),
		NumPorts:        in.GetNumPorts(),
		ClassVersion:    in.GetClassVersion(),
		BaseVersion:     in.GetBaseVersion(),
		SystemImageGUID: in.GetSystemImageGuid(),
		PortGUID:        in.GetPortGuid(),
		Info:            nodeInfoFromProto(in.GetInfo()),
		RawJSON:         in.GetRawJson(),
	}
}

func nodeInfoFromProto(in *repositorypb.NodeInfo) *core.NodeInfo {
	if in == nil {
		return nil
	}

	return &core.NodeInfo{
		ID:           in.GetId(),
		NodeID:       in.GetNodeId(),
		NodeGUID:     in.GetNodeGuid(),
		SerialNumber: in.GetSerialNumber(),
		PartNumber:   in.GetPartNumber(),
		Revision:     in.GetRevision(),
		ProductName:  in.GetProductName(),
		RawJSON:      in.GetRawJson(),
	}
}

func portsFromProto(in []*repositorypb.Port) []core.Port {
	out := make([]core.Port, 0, len(in))

	for _, port := range in {
		if port == nil {
			continue
		}

		out = append(out, portFromProto(port))
	}

	return out
}

func portFromProto(in *repositorypb.Port) core.Port {
	if in == nil {
		return core.Port{}
	}

	return core.Port{
		ID:              in.GetId(),
		LogID:           in.GetLogId(),
		NodeID:          in.GetNodeId(),
		NodeGUID:        in.GetNodeGuid(),
		PortGUID:        in.GetPortGuid(),
		PortNum:         in.GetPortNum(),
		LID:             in.GetLid(),
		LocalPortNum:    in.GetLocalPortNum(),
		PortState:       in.GetPortState(),
		PortPhyState:    in.GetPortPhyState(),
		LinkWidthActive: in.GetLinkWidthActive(),
		LinkSpeedActive: in.GetLinkSpeedActive(),
		RawJSON:         in.GetRawJson(),
	}
}

func mapGRPCError(err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument:
		return core.ErrBadArguments
	case codes.NotFound:
		return core.ErrNotFound
	case codes.FailedPrecondition, codes.Aborted, codes.AlreadyExists:
		return core.ErrConflict
	case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
		return core.ErrUnavailable
	default:
		return err
	}
}

var _ core.Repository = (*Client)(nil)
