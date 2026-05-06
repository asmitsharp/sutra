package mcp

import (
	"context"
)

type Transport interface {
	Send(ctx context.Context, msg *JSONRPCMessage) error
	Recv(ctx context.Context) (*JSONRPCMessage, error)
	Close() error
}
