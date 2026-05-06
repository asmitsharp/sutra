package sse

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/ashmitsharp/sutra/mcp"
)

type SSETransport struct {
	incoming chan *mcp.JSONRPCMessage
	outgoing chan *mcp.JSONRPCMessage
	done     chan struct{}
}

func (t *SSETransport) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", t.handleSSE)
	mux.HandleFunc("POST /message", t.handleMessage)

	return mux
}

func (t *SSETransport) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}
	flusher.Flush()

	for {
		select {
		case msg, ok := <-t.outgoing:
			if !ok {
				return
			}
			data, err := mcp.Encode(msg)
			if err != nil {
				continue
			}
			_, e := fmt.Fprintf(w, "data: %s\n\n", data)
			if e != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (t *SSETransport) handleMessage(w http.ResponseWriter, r *http.Request) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	msg, err := mcp.Decode(data)
	if err != nil {
		http.Error(w, "Failed to decode message", http.StatusBadRequest)
		return
	}

	select {
	case t.incoming <- msg:
	case <-r.Context().Done():
		http.Error(w, "Server busy", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Message received"))
}

func (t *SSETransport) Send(ctx context.Context, msg *mcp.JSONRPCMessage) error {
	select {
	case <-t.done:
		return &mcp.MCPError{
			Code:    mcp.ErrInternal,
			Message: "Transport is closed",
		}
	case t.outgoing <- msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *SSETransport) Recv(ctx context.Context) (*mcp.JSONRPCMessage, error) {
	select {
	case <-t.done:
		return nil, &mcp.MCPError{
			Code:    mcp.ErrInternal,
			Message: "Transport is closed",
		}
	case msg, ok := <-t.incoming:
		if !ok {
			return nil, io.EOF
		}
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *SSETransport) Close() error {
	close(t.done)
	return nil
}

func New() *SSETransport {
	return &SSETransport{
		incoming: make(chan *mcp.JSONRPCMessage, 32),
		outgoing: make(chan *mcp.JSONRPCMessage, 32),
		done:     make(chan struct{}),
	}
}
