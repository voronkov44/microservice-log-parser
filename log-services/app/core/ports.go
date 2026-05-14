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
}

type LogParser interface {
	ParseLog(ctx context.Context, path string) (ParseLogResult, error)
}
