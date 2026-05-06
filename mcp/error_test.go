package mcp

import (
	"testing"
)

func TestMCPError(t *testing.T) {
	err := &MCPError{
		Code:    ErrInvalidRequest,
		Message: "Invalid JSON-RPC message",
	}
	expected := "[Code: -32600] Invalid JSON-RPC message"
	if got := err.Error(); got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
