package parser

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
)

type Client struct {
	log    *slog.Logger
	client parserpb.ParserClient
	conn   *grpc.ClientConn
}

func NewClient(address string, log *slog.Logger) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("new grpc client for parser %s: %w", address, err)
	}

	return &Client{
		log:    log,
		client: parserpb.NewParserClient(conn),
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

func (c *Client) Parse(ctx context.Context, path string) (core.ParsedLog, error) {
	resp, err := c.client.Parse(ctx, &parserpb.ParseRequest{
		Path: path,
	})
	if err != nil {
		return core.ParsedLog{}, mapGRPCError(err)
	}

	return parsedLogFromProto(resp.GetLog()), nil
}

func parsedLogFromProto(in *parserpb.ParsedLog) core.ParsedLog {
	if in == nil {
		return core.ParsedLog{}
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

var _ core.Parser = (*Client)(nil)
