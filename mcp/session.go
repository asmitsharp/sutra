package mcp

import (
	"fmt"
	"sync"
)

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCapability struct{}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

type ClientCapabilities struct {
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// SessionState is an enum representing where we are in the MCP lifecycle
type SessionState int

const (
	StateUninitialized SessionState = iota
	StateInitializing
	StateReady
	StateTerminated
)

func (s SessionState) String() string {
	switch s {
	case StateUninitialized:
		return "Uninitialized"
	case StateInitializing:
		return "Initializing"
	case StateReady:
		return "Ready"
	case StateTerminated:
		return "Terminated"
	default:
		return fmt.Sprintf("Unknown(%d)", int(s))
	}
}

type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type Session struct {
	id    string
	state SessionState
	mu    sync.RWMutex

	clientInfo     ClientInfo
	negotiatedCaps ServerCapabilities
	transport      Transport
}

func NewSession(id string, transport Transport, caps ServerCapabilities) *Session {
	return &Session{
		id:             id,
		transport:      transport,
		negotiatedCaps: caps,
	}
}

// HandleInitialize processes the client's opening move.
// It validates state, records client info, and returns our capabilities.
func (s *Session) HandleInitialize(params InitializeParams) (*InitializeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateUninitialized {
		return nil, &MCPError{
			Code:    ErrInvalidRequest,
			Message: fmt.Sprintf("cannot initialize in state %s", s.state),
		}
	}
	s.clientInfo = params.ClientInfo
	s.state = StateInitializing

	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    s.negotiatedCaps,
		ServerInfo:      ServerInfo{Name: "goagent", Version: "0.1.0"},
	}, nil
}

// HandleInitialized processes the client's confirmation notification.
// After this, the session is fully open for business.
func (s *Session) HandleInitialized() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != StateInitializing {
		return &MCPError{
			Code:    ErrInvalidRequest,
			Message: fmt.Sprintf("received initialized in unexpected state %s", s.state),
		}
	}

	s.state = StateReady
	return nil
}

// RequireReady is the guard to ensure session is ready before processing a tool request
func (s *Session) RequireReady() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.state != StateReady {
		return &MCPError{
			Code:    ErrSessionNotInitialized,
			Message: fmt.Sprintf("session not ready (state: %s)", s.state),
		}
	}
	return nil
}

// Terminate closes the session.
func (s *Session) Terminate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = StateTerminated
}

// State returns a snapshot of the current state (safe to call concurrently)
func (s *Session) State() SessionState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

// ClientInfo returns the recorded client info (safe after Ready state)
func (s *Session) ClientInfo() ClientInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clientInfo
}

func (s *Session) ID() string {
	return s.id
}
