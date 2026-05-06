package mcp

import (
	"context"
	"encoding/json"
	"log"
	"sync"
)

type HandlerFunc func(ctx context.Context, session *Session, params json.RawMessage) (any, error)

type Router struct {
	handlers map[string]HandlerFunc
	mu       sync.RWMutex
}

func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]HandlerFunc),
	}
}

func (r *Router) Handle(method string, h HandlerFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[method] = h
}

func (r *Router) Dispatch(ctx context.Context, session *Session, msg *JSONRPCMessage) {
	isNotification := msg.ID == nil || msg.ID.IsNull()

	if msg.Method == "initialize" {
		var params InitializeParams
		if err := unmarshalParams(msg.Params, &params); err != nil {
			r.sendError(ctx, session, msg.ID, ErrInvalidParams, "invalid initialize params", nil)
			return
		}

		result, err := session.HandleInitialize(params)
		if err != nil {
			r.sendError(ctx, session, msg.ID, errorCode(err), err.Error(), nil)
			return
		}
		r.sendResult(ctx, session, msg.ID, result)
		return
	}

	if msg.Method == "notifications/initialized" {
		if err := session.HandleInitialized(); err != nil {
			log.Printf("session %s: invalid initialized notification: %v", session.ID(), err)
			session.Terminate()
		}
		return
	}

	if err := session.RequireReady(); err != nil {
		if !isNotification {
			r.sendError(ctx, session, msg.ID, ErrSessionNotInitialized, err.Error(), nil)
		}
		return
	}

	r.mu.RLock()
	h, ok := r.handlers[msg.Method]
	r.mu.RUnlock()

	if !ok {
		if !isNotification {
			r.sendError(ctx, session, msg.ID, ErrMethodNotFound, "method not found: "+msg.Method, nil)
		}
		return
	}

	var params json.RawMessage
	if msg.Params != nil {
		params = *msg.Params
	}

	result, err := h(ctx, session, params)

	if isNotification {
		if err != nil {
			log.Printf("session %s: handler error for notification %s: %v", session.ID(), msg.Method, err)
		}
		return
	}

	if err != nil {
		r.sendError(ctx, session, msg.ID, errorCode(err), err.Error(), nil)
		return
	}

	r.sendResult(ctx, session, msg.ID, result)
}

// sendResult encodes a successful response and sends it through the transport.
func (r *Router) sendResult(ctx context.Context, session *Session, id *ID, result any) {
	if id == nil {
		return // safety guard — should never happen for requests
	}

	msg, err := NewResponse(*id, result)
	if err != nil {
		log.Printf("failed to encode response: %v", err)
		return
	}

	if err := session.transport.Send(ctx, msg); err != nil {
		log.Printf("failed to send response: %v", err)
	}
}

// sendError encodes an error response and sends it through the transport.
func (r *Router) sendError(ctx context.Context, session *Session, id *ID, code int, message string, data any) {
	if id == nil {
		return
	}

	var rawData *json.RawMessage
	if data != nil {
		b, _ := json.Marshal(data)
		rd := json.RawMessage(b)
		rawData = &rd
	}

	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    rawData,
		},
	}

	if err := session.transport.Send(ctx, msg); err != nil {
		log.Printf("failed to send error response: %v", err)
	}
}

// errorCode extracts the JSON-RPC error code from an error.
func errorCode(err error) int {
	if mcpErr, ok := err.(*MCPError); ok {
		return mcpErr.Code
	}
	return ErrInternal
}

// unmarshalParams safely unmarshals raw JSON params into a typed struct.
func unmarshalParams(raw *json.RawMessage, dest any) error {
	if raw == nil {
		return &MCPError{Code: ErrInvalidParams, Message: "missing params"}
	}
	if err := json.Unmarshal(*raw, dest); err != nil {
		return &MCPError{Code: ErrInvalidParams, Message: "malformed params: " + err.Error()}
	}
	return nil
}

// Serve reads messages from the session's transport and dispatches them
// to the router. It blocks until an error occurs or the context is cancelled.
func (r *Router) Serve(ctx context.Context, session *Session) error {
	for {
		msg, err := session.transport.Recv(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			session.Terminate()
			return err
		}

		r.Dispatch(ctx, session, msg)
	}
}
