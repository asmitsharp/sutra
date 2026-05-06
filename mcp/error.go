package mcp

import "fmt"

// JSON RPC 2.0 Standard Error Codes
const (
	ErrParse          = -32700
	ErrInvalidRequest = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603

	ErrCapabilityNotSupported = -32000
	ErrSessionNotInitialized  = -32001
)

type MCPError struct {
	Code    int
	Message string
	Data    any
}

func (e *MCPError) Error() string {
	return fmt.Sprintf("[Code: %d] %s", e.Code, e.Message)
}
