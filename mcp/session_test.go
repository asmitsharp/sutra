package mcp

import (
	"context"
	"testing"
)

// mockTransport for session testing
type mockTransport struct{}

func (m *mockTransport) Send(ctx context.Context, msg *JSONRPCMessage) error { return nil }
func (m *mockTransport) Recv(ctx context.Context) (*JSONRPCMessage, error)   { return nil, nil }
func (m *mockTransport) Close() error                                        { return nil }

func TestSessionStateTransitions(t *testing.T) {
	caps := ServerCapabilities{}
	session := NewSession("test-session", &mockTransport{}, caps)

	if session.State() != StateUninitialized {
		t.Errorf("expected StateUninitialized, got %s", session.State())
	}
	if session.ID() != "test-session" {
		t.Errorf("expected id 'test-session', got %s", session.ID())
	}

	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0",
		},
	}

	// 1. Uninitialized -> Initializing
	res, err := session.HandleInitialize(params)
	if err != nil {
		t.Fatalf("HandleInitialize failed: %v", err)
	}
	if res.ProtocolVersion != "2024-11-05" {
		t.Errorf("unexpected protocol version: %s", res.ProtocolVersion)
	}
	if session.State() != StateInitializing {
		t.Errorf("expected StateInitializing, got %s", session.State())
	}

	// Verify client info is set
	if session.ClientInfo().Name != "test-client" {
		t.Errorf("expected test-client, got %s", session.ClientInfo().Name)
	}

	// Calling HandleInitialize again should fail
	_, err = session.HandleInitialize(params)
	if err == nil {
		t.Error("expected error calling HandleInitialize twice")
	}

	// Calling RequireReady before Initialized should fail
	if err := session.RequireReady(); err == nil {
		t.Error("RequireReady should fail before HandleInitialized")
	}

	// 2. Initializing -> Ready
	if err := session.HandleInitialized(); err != nil {
		t.Fatalf("HandleInitialized failed: %v", err)
	}
	if session.State() != StateReady {
		t.Errorf("expected StateReady, got %s", session.State())
	}

	// Calling HandleInitialized again should fail
	if err := session.HandleInitialized(); err == nil {
		t.Error("expected error calling HandleInitialized twice")
	}

	// RequireReady should now succeed
	if err := session.RequireReady(); err != nil {
		t.Errorf("RequireReady failed: %v", err)
	}

	// 3. Ready -> Terminated
	session.Terminate()
	if session.State() != StateTerminated {
		t.Errorf("expected StateTerminated, got %s", session.State())
	}

	// RequireReady should fail after termination
	if err := session.RequireReady(); err == nil {
		t.Error("RequireReady should fail after termination")
	}
}

func TestSessionStateString(t *testing.T) {
	tests := []struct {
		state SessionState
		want  string
	}{
		{StateUninitialized, "Uninitialized"},
		{StateInitializing, "Initializing"},
		{StateReady, "Ready"},
		{StateTerminated, "Terminated"},
		{SessionState(99), "Unknown(99)"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("StateString() = %v, want %v", got, tt.want)
		}
	}
}
