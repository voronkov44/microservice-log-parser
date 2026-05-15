package core

import "context"

type Topology interface {
	Ping(context.Context) error
	GetTopology(ctx context.Context, logID int64) (TopologyResult, error)
}

type Repository interface {
	Ping(context.Context) error

	GetTopologyData(ctx context.Context, logID int64) (TopologyData, error)
}
