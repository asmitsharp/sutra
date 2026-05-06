package stdio

import (
	"context"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/ashmitsharp/sutra/mcp"
)

func TestStdioRoundTrip(t *testing.T) {
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	client := New(clientReader, clientWriter)
	server := New(serverReader, serverWriter)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var id mcp.ID
	id.UnmarshalJSON([]byte(`1`))
	req, err := mcp.NewRequest(id, "ping", map[string]string{})
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	go func() {
		err := client.Send(ctx, req)
		if err != nil {
			t.Errorf("Client send failed: %v", err)
		}
	}()

	msg, err := server.Recv(ctx)
	if err != nil {
		t.Fatalf("Server receive failed: %v", err)
	}

	if !reflect.DeepEqual(msg, req) {
		t.Errorf("Received message %+v, want %+v", msg, req)
	}

	// Test EOF
	clientWriter.Close()
	_, err = server.Recv(ctx)
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}

	// Test close
	if err := client.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestStdioSendContextCanceled(t *testing.T) {
	client := New(nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	var id mcp.ID
	id.UnmarshalJSON([]byte(`1`))
	req, _ := mcp.NewRequest(id, "ping", nil)

	err := client.Send(ctx, req)
	if err != context.Canceled {
		t.Errorf("Expected context canceled, got %v", err)
	}
}

type failWriter struct{}
func (f *failWriter) Write(p []byte) (n int, err error) { return 0, io.ErrClosedPipe }

func TestStdioSendErrors(t *testing.T) {
	client := New(nil, &failWriter{})
	var id mcp.ID
	id.UnmarshalJSON([]byte(`1`))
	req, _ := mcp.NewRequest(id, "ping", nil)
	
	err := client.Send(context.Background(), req)
	if err == nil {
		t.Error("expected error on failed write")
	}
}

type errReader struct{}
func (e *errReader) Read(p []byte) (n int, err error) { return 0, io.ErrUnexpectedEOF }

func TestStdioRecvErrors(t *testing.T) {
    client := New(&errReader{}, nil)
    _, err := client.Recv(context.Background())
    if err == nil || err == io.EOF {
        t.Error("expected scanner error")
    }
}
