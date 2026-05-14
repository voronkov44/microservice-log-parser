package core

import "context"

type Pinger interface {
	Ping(context.Context) error
}

type Parser interface {
	Pinger
	Parse(ctx context.Context, path string) (ParsedLog, error)
}

type Repository interface {
	Pinger

	CreateLog(ctx context.Context, filePath string) (int64, error)
	SaveParsedLog(ctx context.Context, logID int64, parsed ParsedLog) (SaveParsedLogResult, error)
	FailLog(ctx context.Context, logID int64, errorText string) error

	GetLog(ctx context.Context, logID int64) (Log, error)
	GetNode(ctx context.Context, nodeID int64) (Node, error)
	GetPortsByNode(ctx context.Context, nodeID int64) ([]Port, error)

	GetNodesByLog(ctx context.Context, logID int64) ([]Node, error)
	GetPortsByLog(ctx context.Context, logID int64) ([]Port, error)
}

type TopologyProvider interface {
	Pinger
	GetTopology(ctx context.Context, logID int64) (Topology, error)
}

type LogParser interface {
	ParseLog(ctx context.Context, path string) (ParseLogResult, error)
}

type TopologyViewer interface {
	GetTopology(ctx context.Context, logID int64) (Topology, error)
}

type LogReader interface {
	GetLog(ctx context.Context, logID int64) (Log, error)
	GetNode(ctx context.Context, nodeID int64) (Node, error)
	GetPortsByNode(ctx context.Context, nodeID int64) ([]Port, error)

	GetNodesByLog(ctx context.Context, logID int64) ([]Node, error)
	GetPortsByLog(ctx context.Context, logID int64) ([]Port, error)
}
