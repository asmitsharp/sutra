package stdio

import (
	"bufio"
	"context"
	"io"
	"sync"

	"github.com/ashmitsharp/sutra/mcp"
)

type StdioTransport struct {
	scanner *bufio.Scanner
	writer  *bufio.Writer
	mu      sync.Mutex
	done    chan struct{}
}

func New(r io.Reader, w io.Writer) *StdioTransport {
	return &StdioTransport{
		scanner: bufio.NewScanner(r),
		writer:  bufio.NewWriter(w),
		done:    make(chan struct{}),
	}
}

// Send encodes and sends a message to the transport.
func (t *StdioTransport) Send(ctx context.Context, msg *mcp.JSONRPCMessage) error {
	encodedData, err := mcp.Encode(msg)
	if err != nil {
		return &mcp.MCPError{
			Code:    mcp.ErrInternal,
			Message: "Failed to encode message",
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.writer.Write(encodedData); err != nil {
		return &mcp.MCPError{
			Code:    mcp.ErrInternal,
			Message: "Failed to write encoded data",
		}
	}

	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return &mcp.MCPError{
			Code:    mcp.ErrInternal,
			Message: "Failed to write newline",
		}
	}

	if err := t.writer.Flush(); err != nil {
		return &mcp.MCPError{
			Code:    mcp.ErrInternal,
			Message: "Failed to flush writer",
		}
	}

	return nil

}

// Recv reads and decodes a message from the transport.
func (t *StdioTransport) Recv(ctx context.Context) (*mcp.JSONRPCMessage, error) {
	if !t.scanner.Scan() {

		// scanner stopped

		if err := t.scanner.Err(); err != nil {
			return nil, err
		}

		// no error means EOF
		return nil, io.EOF
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	data := t.scanner.Bytes()

	msg, err := mcp.Decode(data)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// Close closes the transport.
func (t *StdioTransport) Close() error {
	close(t.done)
	return nil
}
