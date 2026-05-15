package core

import "context"

type Repository interface {
	Ping(context.Context) error

	CreateLog(ctx context.Context, filePath string) (int64, error)
	SaveParsedLog(ctx context.Context, logID int64, parsed ParsedLog) (SaveParsedLogResult, error)
	FailLog(ctx context.Context, logID int64, errorText string) error

	GetLog(ctx context.Context, logID int64) (Log, error)
	GetNode(ctx context.Context, nodeID int64) (Node, error)
	GetPortsByNode(ctx context.Context, nodeID int64) ([]Port, error)

	GetNodesByLog(ctx context.Context, logID int64) ([]Node, error)
	GetPortsByLog(ctx context.Context, logID int64) ([]Port, error)
	GetTopologyData(ctx context.Context, logID int64) (TopologyData, error)
}

type DB interface {
	Ping(context.Context) error

	CreateLog(ctx context.Context, filePath string) (int64, error)
	SaveParsedLog(ctx context.Context, logID int64, parsed ParsedLog) (SaveParsedLogResult, error)
	FailLog(ctx context.Context, logID int64, errorText string) error

	GetLog(ctx context.Context, logID int64) (Log, error)
	GetNode(ctx context.Context, nodeID int64) (Node, error)
	GetPortsByNode(ctx context.Context, nodeID int64) ([]Port, error)

	GetNodesByLog(ctx context.Context, logID int64) ([]Node, error)
	GetPortsByLog(ctx context.Context, logID int64) ([]Port, error)
	GetTopologyData(ctx context.Context, logID int64) (TopologyData, error)
}
