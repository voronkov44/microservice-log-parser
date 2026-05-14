package core

import "context"

type Topology interface {
	Ping(context.Context) error
	GetTopology(ctx context.Context, logID int64) (TopologyResult, error)
}

type Repository interface {
	Ping(context.Context) error

	GetLog(ctx context.Context, logID int64) (Log, error)
	GetNodesByLog(ctx context.Context, logID int64) ([]Node, error)
	GetPortsByLog(ctx context.Context, logID int64) ([]Port, error)
}
