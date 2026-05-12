package core

import "context"

type Repository interface {
	Ping(context.Context) error
}

type DB interface {
	Ping(context.Context) error
}
