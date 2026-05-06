package mcp

import "encoding/json"

type ID struct {
	raw json.RawMessage
}

type JSONRPCError struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    *json.RawMessage `json:"data,omitempty"`
}

type JSONRPCMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *ID              `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  *json.RawMessage `json:"params,omitempty"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError    `json:"error,omitempty"`
}

// UnmarshalJSON — store raw bytes without parsing the type yet
func (id *ID) UnmarshalJSON(b []byte) error {
	id.raw = append(id.raw[:0], b...)
	return nil
}

// MarshalJSON — emit the raw value as-is
func (id *ID) MarshalJSON() ([]byte, error) {
	if id.raw == nil {
		return []byte("null"), nil
	}
	return id.raw, nil
}

func (id *ID) IsNull() bool {
	return len(id.raw) == 0 || string(id.raw) == "null"
}

func (id *ID) IsString() bool {
	return len(id.raw) > 0 && id.raw[0] == '"' && id.raw[len(id.raw)-1] == '"'
}

func (id *ID) IsNumber() bool {
	if id.IsNull() {
		return false
	}
	var n float64
	return json.Unmarshal(id.raw, &n) == nil
}

func (id *ID) StringValue() string {
	var s string
	json.Unmarshal(id.raw, &s)
	return s
}

func (id *ID) NumberValue() float64 {
	var n float64
	json.Unmarshal(id.raw, &n)
	return n
}

// Encode serializes a JSONRPCMessage to bytes
func Encode(msg *JSONRPCMessage) ([]byte, error) {
	return json.Marshal(msg)
}

// Decode parses JSONRPCMessage from bytes
func Decode(data []byte) (*JSONRPCMessage, error) {
	var msg JSONRPCMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	if msg.Method != "" {
		// Request or Notification
		return &msg, nil
	}

	if msg.Result != nil || msg.Error != nil {
		// Response
		return &msg, nil
	}

	return nil, &MCPError{
		Code:    ErrInvalidRequest,
		Message: "Invalid JSON-RPC message",
	}
}

func NewRequest(id ID, method string, params any) (*JSONRPCMessage, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(data)
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  &raw,
	}, nil
}

func NewResponse(id ID, result any) (*JSONRPCMessage, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(data)
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      &id,
		Result:  &raw,
	}, nil
}

func NewNotification(method string, params any) (*JSONRPCMessage, error) {
	data, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(data)
	return &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
		Params:  &raw,
	}, nil
}
