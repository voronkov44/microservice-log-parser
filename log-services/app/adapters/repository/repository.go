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

	"github.com/voronkov44/microservice-log-parser/log-services/app/core"
	parserpb "github.com/voronkov44/microservice-log-parser/log-services/proto/parser"
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
