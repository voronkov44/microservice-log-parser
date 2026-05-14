package core

import "context"

type Parser interface {
	Ping(ctx context.Context) error
	Parse(ctx context.Context, path string) (ParsedLog, error)
}

type Engine interface {
	Parse(ctx context.Context, path string) (ParsedLog, error)
}
