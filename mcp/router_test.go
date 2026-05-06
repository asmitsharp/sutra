package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

type captureTransport struct {
	sent []*JSONRPCMessage
	recv chan *JSONRPCMessage
}

func (t *captureTransport) Send(ctx context.Context, msg *JSONRPCMessage) error {
	t.sent = append(t.sent, msg)
	return nil
}

func (t *captureTransport) Recv(ctx context.Context) (*JSONRPCMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg := <-t.recv:
		return msg, nil
	}
}

func (t *captureTransport) Close() error { return nil }

func TestRouterDispatch(t *testing.T) {
	router := NewRouter()
	transport := &captureTransport{
		recv: make(chan *JSONRPCMessage, 10),
	}
	session := NewSession("test", transport, ServerCapabilities{})

	// 1. Initialize
	var id ID
	id.UnmarshalJSON([]byte(`1`))
	initMsg, _ := NewRequest(id, "initialize", InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      ClientInfo{Name: "test", Version: "1.0"},
	})
	router.Dispatch(context.Background(), session, initMsg)

	if len(transport.sent) != 1 {
		t.Fatalf("expected 1 response, got %d", len(transport.sent))
	}
	res := transport.sent[0]
	if res.Error != nil {
		t.Fatalf("unexpected error response: %+v", res.Error)
	}

	// 2. Notifications/initialized
	initializedNotif, _ := NewNotification("notifications/initialized", nil)
	router.Dispatch(context.Background(), session, initializedNotif)

	if session.State() != StateReady {
		t.Fatalf("expected state ready, got %v", session.State())
	}

	// 3. Normal Route
	router.Handle("test/method", func(ctx context.Context, s *Session, params json.RawMessage) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	var reqId ID
	reqId.UnmarshalJSON([]byte(`2`))
	reqMsg, _ := NewRequest(reqId, "test/method", nil)
	router.Dispatch(context.Background(), session, reqMsg)

	if len(transport.sent) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(transport.sent))
	}
	res2 := transport.sent[1]
	if string(*res2.Result) != `{"status":"ok"}` {
		t.Errorf("unexpected result: %s", string(*res2.Result))
	}

	// 4. Missing method
	var reqId3 ID
	reqId3.UnmarshalJSON([]byte(`3`))
	missingMsg, _ := NewRequest(reqId3, "missing", nil)
	router.Dispatch(context.Background(), session, missingMsg)

	res3 := transport.sent[2]
	if res3.Error == nil || res3.Error.Code != ErrMethodNotFound {
		t.Errorf("expected ErrMethodNotFound, got %+v", res3.Error)
	}
}

func TestRouterServe(t *testing.T) {
	router := NewRouter()
	transport := &captureTransport{
		recv: make(chan *JSONRPCMessage, 1),
	}
	session := NewSession("test", transport, ServerCapabilities{})

	ctx, cancel := context.WithCancel(context.Background())

	// Start serve
	errCh := make(chan error)
	go func() {
		errCh <- router.Serve(ctx, session)
	}()

	var id ID
	id.UnmarshalJSON([]byte(`1`))
	initMsg, _ := NewRequest(id, "initialize", InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo:      ClientInfo{Name: "test", Version: "1.0"},
	})
	transport.recv <- initMsg

	// Wait a bit to ensure it processes
	time.Sleep(50 * time.Millisecond)

	cancel()
	err := <-errCh
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

type failTransport struct{}
func (t *failTransport) Send(ctx context.Context, msg *JSONRPCMessage) error { return context.Canceled }
func (t *failTransport) Recv(ctx context.Context) (*JSONRPCMessage, error) { return nil, context.Canceled }
func (t *failTransport) Close() error { return nil }

func TestRouterCoverage(t *testing.T) {
	router := NewRouter()
	transport := &captureTransport{
		recv: make(chan *JSONRPCMessage, 10),
	}
	session := NewSession("test", transport, ServerCapabilities{})
	ctx := context.Background()
	
	router.sendResult(ctx, session, nil, "test")
	
	var id ID
	id.UnmarshalJSON([]byte(`1`))
	router.sendResult(ctx, session, &id, make(chan int))
	
	router.sendError(ctx, session, nil, ErrInternal, "error", nil)

	var dest InitializeParams
	err := unmarshalParams(nil, &dest)
	if err == nil || err.(*MCPError).Code != ErrInvalidParams {
		t.Error("expected ErrInvalidParams for nil raw")
	}
	
	badRaw := json.RawMessage([]byte("{bad"))
	err = unmarshalParams(&badRaw, &dest)
	if err == nil || err.(*MCPError).Code != ErrInvalidParams {
		t.Error("expected ErrInvalidParams for bad raw")
	}
	
	if c := errorCode(context.Canceled); c != ErrInternal {
		t.Errorf("expected ErrInternal, got %d", c)
	}

	router.Handle("test/notif", func(ctx context.Context, s *Session, params json.RawMessage) (any, error) {
		return nil, &MCPError{Code: ErrInternal, Message: "test error"}
	})
	notifMsg, _ := NewNotification("test/notif", nil)
	session.HandleInitialize(InitializeParams{ClientInfo: ClientInfo{Name: "test"}})
	session.HandleInitialized()
	router.Dispatch(ctx, session, notifMsg) 
	
	initMsgBad, _ := NewRequest(id, "initialize", nil)
	initMsgBad.Params = &badRaw
	router.Dispatch(ctx, session, initMsgBad)
	
	missingNotif, _ := NewNotification("missing/notif", nil)
	router.Dispatch(ctx, session, missingNotif)

    failTrans := &failTransport{}
	failSess := NewSession("f", failTrans, ServerCapabilities{})
    router.sendResult(ctx, failSess, &id, "ok")
	router.sendError(ctx, failSess, &id, ErrInternal, "err", nil)
	
	router.Serve(ctx, failSess)
}
