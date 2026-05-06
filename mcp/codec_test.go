package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    *JSONRPCMessage
		wantErr bool
	}{
		{"request", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, &JSONRPCMessage{JSONRPC: "2.0", ID: &ID{raw: []byte(`1`)}, Method: "tools/list"}, false},
		{"notification", `{"jsonrpc":"2.0","method":"initialized"}`, &JSONRPCMessage{JSONRPC: "2.0", Method: "initialized"}, false},
		{"response", `{"jsonrpc":"2.0","id":1,"result":{}}`, &JSONRPCMessage{JSONRPC: "2.0", ID: &ID{raw: []byte(`1`)}, Result: rawMsg(`{}`)}, false},
		{"error response", `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"not found"}}`, &JSONRPCMessage{JSONRPC: "2.0", ID: &ID{raw: []byte(`1`)}, Error: &JSONRPCError{Code: -32601, Message: "not found"}}, false},
		{"string id", `{"jsonrpc":"2.0","id":"abc","method":"ping"}`, &JSONRPCMessage{JSONRPC: "2.0", ID: &ID{raw: []byte(`"abc"`)}, Method: "ping"}, false},
		{"null id", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, &JSONRPCMessage{JSONRPC: "2.0", ID: nil, Method: "ping"}, false},
		{"invalid json", `{broken`, nil, true},
		{"missing fields", `{"jsonrpc":"2.0"}`, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode([]byte(tc.input))
			if (err != nil) != tc.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if tc.wantErr {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Decode() got = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func rawMsg(s string) *json.RawMessage {
	r := json.RawMessage(s)
	return &r
}

func TestIDMethods(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		isNull      bool
		isString    bool
		isNumber    bool
		stringVal   string
		numberVal   float64
		marshalWant string
	}{
		{"number", "123", false, false, true, "", 123, "123"},
		{"string", `"abc"`, false, true, false, "abc", 0, `"abc"`},
		{"null", `null`, true, false, false, "", 0, `null`},
		{"empty", ``, true, false, false, "", 0, `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ID
			if tt.input != "" {
				err := id.UnmarshalJSON([]byte(tt.input))
				if err != nil {
					t.Fatalf("UnmarshalJSON failed: %v", err)
				}
			}

			if got := id.IsNull(); got != tt.isNull {
				t.Errorf("IsNull() = %v, want %v", got, tt.isNull)
			}
			if got := id.IsString(); got != tt.isString {
				t.Errorf("IsString() = %v, want %v", got, tt.isString)
			}
			if got := id.IsNumber(); got != tt.isNumber {
				t.Errorf("IsNumber() = %v, want %v", got, tt.isNumber)
			}
			if got := id.StringValue(); got != tt.stringVal {
				t.Errorf("StringValue() = %v, want %v", got, tt.stringVal)
			}
			if got := id.NumberValue(); got != tt.numberVal {
				t.Errorf("NumberValue() = %v, want %v", got, tt.numberVal)
			}
			marshaled, err := id.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
			if string(marshaled) != tt.marshalWant {
				t.Errorf("MarshalJSON() = %s, want %s", string(marshaled), tt.marshalWant)
			}
		})
	}
}

func TestEncode(t *testing.T) {
	msg := &JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "test",
	}
	data, err := Encode(msg)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	expected := `{"jsonrpc":"2.0","method":"test"}`
	if string(data) != expected {
		t.Errorf("Encode() = %s, want %s", string(data), expected)
	}
}

func TestNewConstructors(t *testing.T) {
	var id ID
	id.UnmarshalJSON([]byte(`1`))

	req, err := NewRequest(id, "testReq", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	if req.JSONRPC != "2.0" || req.Method != "testReq" || string(*req.Params) != `{"foo":"bar"}` {
		t.Errorf("Unexpected request: %+v", req)
	}
	_, err = NewRequest(id, "testReq", make(chan int)) // trigger json.Marshal error
	if err == nil {
		t.Error("Expected error on invalid param for NewRequest")
	}

	res, err := NewResponse(id, map[string]string{"result": "ok"})
	if err != nil {
		t.Fatalf("NewResponse failed: %v", err)
	}
	if res.JSONRPC != "2.0" || string(*res.Result) != `{"result":"ok"}` {
		t.Errorf("Unexpected response: %+v", res)
	}
	_, err = NewResponse(id, make(chan int)) // trigger json.Marshal error
	if err == nil {
		t.Error("Expected error on invalid param for NewResponse")
	}

	notif, err := NewNotification("testNotif", map[string]string{"event": "start"})
	if err != nil {
		t.Fatalf("NewNotification failed: %v", err)
	}
	if notif.JSONRPC != "2.0" || notif.Method != "testNotif" || string(*notif.Params) != `{"event":"start"}` {
		t.Errorf("Unexpected notification: %+v", notif)
	}
	_, err = NewNotification("testNotif", make(chan int)) // trigger json.Marshal error
	if err == nil {
		t.Error("Expected error on invalid param for NewNotification")
	}
}
