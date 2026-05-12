package core

import "context"

type Topology interface {
	Ping(context.Context) error
}
