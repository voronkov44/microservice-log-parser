package repository

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	repositorypb "github.com/voronkov44/microservice-log-parser/log-services/proto/repository"
	"github.com/voronkov44/microservice-log-parser/log-services/topology/core"
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

func (c *Client) GetNodesByLog(ctx context.Context, logID int64) ([]core.Node, error) {
	resp, err := c.client.GetNodesByLog(ctx, &repositorypb.GetNodesByLogRequest{
		LogId: logID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	nodes := make([]core.Node, 0, len(resp.GetNodes()))
	for _, node := range resp.GetNodes() {
		if node == nil {
			continue
		}

		nodes = append(nodes, nodeFromProto(node))
	}

	return nodes, nil
}

func (c *Client) GetPortsByLog(ctx context.Context, logID int64) ([]core.Port, error) {
	resp, err := c.client.GetPortsByLog(ctx, &repositorypb.GetPortsByLogRequest{
		LogId: logID,
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	ports := make([]core.Port, 0, len(resp.GetPorts()))
	for _, port := range resp.GetPorts() {
		if port == nil {
			continue
		}

		ports = append(ports, portFromProto(port))
	}

	return ports, nil
}

func logFromProto(in *repositorypb.Log) core.Log {
	if in == nil {
		return core.Log{}
	}

	return core.Log{
		ID:             in.GetId(),
		FilePath:       in.GetFilePath(),
		Status:         logStatusFromProto(in.GetStatus()),
		NodesCount:     in.GetNodesCount(),
		PortsCount:     in.GetPortsCount(),
		Error:          in.GetError(),
		UploadedAtUnix: in.GetUploadedAtUnix(),
		ParsedAtUnix:   in.GetParsedAtUnix(),
	}
}

func logStatusFromProto(in repositorypb.LogStatus) core.LogStatus {
	switch in {
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

func nodeFromProto(in *repositorypb.NodeDetails) core.Node {
	node := core.Node{
		ID:       in.GetId(),
		LogID:    in.GetLogId(),
		NodeGUID: in.GetNodeGuid(),
		NodeDesc: in.GetNodeDesc(),
		NodeType: in.GetNodeType(),
		NodeKind: in.GetNodeKind(),
		NumPorts: in.GetNumPorts(),
	}

	if in.GetInfo() != nil {
		node.Info = &core.NodeInfo{
			SerialNumber: in.GetInfo().GetSerialNumber(),
			ProductName:  in.GetInfo().GetProductName(),
		}
	}

	return node
}

func portFromProto(in *repositorypb.Port) core.Port {
	return core.Port{
		ID:              in.GetId(),
		LogID:           in.GetLogId(),
		NodeID:          in.GetNodeId(),
		NodeGUID:        in.GetNodeGuid(),
		PortGUID:        in.GetPortGuid(),
		PortNum:         in.GetPortNum(),
		PortState:       in.GetPortState(),
		LinkWidthActive: in.GetLinkWidthActive(),
		LinkSpeedActive: in.GetLinkSpeedActive(),
	}
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

var _ core.Repository = (*Client)(nil)
