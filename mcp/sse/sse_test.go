package sse

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ashmitsharp/sutra/mcp"
)

func TestSSEReceivesMessage(t *testing.T) {
	transport := New()
	srv := httptest.NewServer(transport.Handler())
	defer srv.Close()

	// 1. Post a message
	var id mcp.ID
	id.UnmarshalJSON([]byte(`1`))
	reqMsg, _ := mcp.NewRequest(id, "ping", nil)
	data, _ := json.Marshal(reqMsg)

	resp, err := http.Post(srv.URL+"/message", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Failed to post message: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected 202 Accepted, got %d", resp.StatusCode)
	}

	// 2. Read from Recv
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	recvMsg, err := transport.Recv(ctx)
	if err != nil {
		t.Fatalf("Recv failed: %v", err)
	}
	if recvMsg.Method != "ping" {
		t.Errorf("Expected method 'ping', got '%s'", recvMsg.Method)
	}
}

func TestSSESendsMessage(t *testing.T) {
	transport := New()
	srv := httptest.NewServer(transport.Handler())
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Start streaming from /events
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/events", nil)
	client := &http.Client{}

	respCh := make(chan *http.Response)
	errCh := make(chan error)
	go func() {
		resp, err := client.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		respCh <- resp
	}()

	// Wait for server to start streaming
	var resp *http.Response
	select {
	case resp = <-respCh:
	case err := <-errCh:
		t.Fatalf("Failed to connect to /events: %v", err)
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for /events connection")
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", resp.StatusCode)
	}

	// 2. Send a message
	var id mcp.ID
	id.UnmarshalJSON([]byte(`2`))
	msg, _ := mcp.NewResponse(id, map[string]string{"result": "ok"})

	err := transport.Send(context.Background(), msg)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// 3. Read the stream
	buf := make([]byte, 1024)
	n, err := resp.Body.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("Failed to read from stream: %v", err)
	}
	output := string(buf[:n])

	expectedPrefix := "data: {"
	if len(output) < len(expectedPrefix) || output[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Unexpected SSE output: %q", output)
	}
}

func TestSSEContextCancellationsAndClose(t *testing.T) {
	transport := New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Immediately cancel

	var id mcp.ID
	id.UnmarshalJSON([]byte(`1`))
	msg, _ := mcp.NewRequest(id, "ping", nil)

	err := transport.Send(ctx, msg)
	if err != context.Canceled {
		t.Errorf("Expected context canceled, got %v", err)
	}

	_, err = transport.Recv(ctx)
	if err != context.Canceled {
		t.Errorf("Expected context canceled, got %v", err)
	}

	transport.Close()

	// Not testing Send after close because t.outgoing is buffered (size 32)
	// and a select with multiple ready cases (closed t.done vs non-full t.outgoing)
	// is non-deterministic in Go.

	_, err = transport.Recv(context.Background())
	if err == nil {
		t.Error("Expected error after close, got nil")
	}
}
