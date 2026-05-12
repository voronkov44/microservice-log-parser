package core

import "context"

type Parser interface {
	Ping(ctx context.Context) error
}
