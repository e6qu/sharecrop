package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

// FuzzHandleRaw drives the JSON-RPC transport with arbitrary bodies against the
// in-memory fake services. Whatever the input, handling must not panic, and an
// emitted payload must be well-formed JSON (the transport promises a valid
// JSON-RPC response or no response at all).
func FuzzHandleRaw(f *testing.F) {
	seeds := []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":null,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":1,"result":{}}`,
		`[{"jsonrpc":"2.0","id":1,"method":"tools/list"},{"jsonrpc":"2.0","method":"x"}]`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"sharecrop.list_tasks","arguments":{}}}`,
		`[]`,
		``,
		`{`,
		`"a string"`,
		`123`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	subject := testSubject(&testing.T{})
	scopes := allScopes()

	f.Fuzz(func(t *testing.T, body []byte) {
		server := NewServer(fakeServices{})
		result := server.HandleRaw(context.Background(), subject, scopes, body)
		if !result.HasResponse {
			if len(result.Payload) != 0 {
				t.Fatalf("no-response result carried a payload: %q", result.Payload)
			}
			return
		}
		if !json.Valid(result.Payload) {
			t.Fatalf("transport emitted invalid JSON for input %q: %q", body, result.Payload)
		}
	})
}
