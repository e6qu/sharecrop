package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestHandleRawSingleRequest(t *testing.T) {
	server := NewServer(fakeServices{})
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	if !result.HasResponse {
		t.Fatalf("expected a response")
	}
	if !strings.Contains(string(result.Payload), "sharecrop.list_tasks") {
		t.Fatalf("payload missing tools: %s", string(result.Payload))
	}
}

func TestHandleRawInitializeSetsSession(t *testing.T) {
	server := NewServer(fakeServices{})
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if result.SessionID == "" {
		t.Fatalf("expected a session id on initialize")
	}
}

func TestHandleRawNotificationHasNoResponse(t *testing.T) {
	server := NewServer(fakeServices{})
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	if result.HasResponse {
		t.Fatalf("notification should produce no response")
	}
}

func TestHandleRawNullIDIsRequest(t *testing.T) {
	server := NewServer(fakeServices{})
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(`{"jsonrpc":"2.0","id":null,"method":"ping"}`))
	if !result.HasResponse {
		t.Fatalf("id:null request should produce a response")
	}
	var response Response
	if err := json.Unmarshal(result.Payload, &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if string(response.ID) != "null" {
		t.Fatalf("id = %s, want null", string(response.ID))
	}
}

func TestHandleRawClientResponseHasNoResponse(t *testing.T) {
	server := NewServer(fakeServices{})
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
	if result.HasResponse {
		t.Fatalf("client response should not be dispatched")
	}
}

func TestHandleRawBatchReturnsArray(t *testing.T) {
	server := NewServer(fakeServices{})
	body := `[{"jsonrpc":"2.0","id":1,"method":"tools/list"},{"jsonrpc":"2.0","id":2,"method":"ping"}]`
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(body))
	if !result.HasResponse {
		t.Fatalf("expected a batch response")
	}
	var responses []Response
	if err := json.Unmarshal(result.Payload, &responses); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	if len(responses) != 2 {
		t.Fatalf("batch response count = %d, want 2", len(responses))
	}
}

func TestHandleRawBatchDropsNotifications(t *testing.T) {
	server := NewServer(fakeServices{})
	body := `[{"jsonrpc":"2.0","method":"notifications/initialized"},{"jsonrpc":"2.0","id":2,"method":"ping"}]`
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(body))
	var responses []Response
	if err := json.Unmarshal(result.Payload, &responses); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	if len(responses) != 1 {
		t.Fatalf("batch response count = %d, want 1 (notification dropped)", len(responses))
	}
}

func TestHandleRawSeriesTool(t *testing.T) {
	server := NewServer(fakeServices{})
	result := server.HandleRaw(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"sharecrop.list_task_series","arguments":{}}}`))
	if !strings.Contains(string(result.Payload), "series") {
		t.Fatalf("list_task_series payload missing series key: %s", string(result.Payload))
	}
}
