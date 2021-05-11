package protocol

import "context"

type Operation interface {
	Execute(ctx context.Context)
}

type Operation2 func()
