package mcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
)

// RawResult is the transport-agnostic outcome of handling a raw JSON-RPC body.
// A request or batch produces a JSON Payload; a notification-only body produces
// no payload (HasResponse is false), which an HTTP transport answers with 202.
type RawResult struct {
	Payload     []byte
	HasResponse bool
	SessionID   string
}

// HandleRaw parses a single JSON-RPC message or a batch array and dispatches
// each through Handle. It is shared by the HTTP and stdio transports.
func (server Server) HandleRaw(ctx context.Context, subject auth.UserSubject, scopes agent.ScopeSet, body []byte) RawResult {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return singleResponse(errorResponse(json.RawMessage("null"), codeInvalidRequest, "request body is empty"))
	}

	if trimmed[0] == '[' {
		return server.handleBatch(ctx, subject, scopes, trimmed)
	}

	var request Request
	if err := json.Unmarshal(trimmed, &request); err != nil {
		return singleResponse(errorResponse(json.RawMessage("null"), codeParseError, "request body is not valid JSON-RPC"))
	}
	if isNotification(request) {
		return RawResult{HasResponse: false}
	}

	result := singleResponse(server.Handle(ctx, subject, scopes, request))
	if request.Method == "initialize" {
		result.SessionID = newSessionID()
	}
	return result
}

func (server Server) handleBatch(ctx context.Context, subject auth.UserSubject, scopes agent.ScopeSet, body []byte) RawResult {
	var requests []Request
	if err := json.Unmarshal(body, &requests); err != nil {
		return singleResponse(errorResponse(json.RawMessage("null"), codeParseError, "batch is not valid JSON-RPC"))
	}
	if len(requests) == 0 {
		return singleResponse(errorResponse(json.RawMessage("null"), codeInvalidRequest, "batch must not be empty"))
	}

	responses := make([]Response, 0, len(requests))
	for index := range requests {
		if isNotification(requests[index]) {
			continue
		}
		responses = append(responses, server.Handle(ctx, subject, scopes, requests[index]))
	}
	if len(responses) == 0 {
		return RawResult{HasResponse: false}
	}

	encoded, err := json.Marshal(responses)
	if err != nil {
		return singleResponse(errorResponse(json.RawMessage("null"), codeInternalError, "failed to encode batch response"))
	}
	return RawResult{Payload: encoded, HasResponse: true}
}

func singleResponse(response Response) RawResult {
	encoded, err := json.Marshal(response)
	if err != nil {
		fallback, _ := json.Marshal(errorResponse(json.RawMessage("null"), codeInternalError, "failed to encode response"))
		return RawResult{Payload: fallback, HasResponse: true}
	}
	return RawResult{Payload: encoded, HasResponse: true}
}

func isNotification(request Request) bool {
	return len(request.ID) == 0 || string(request.ID) == "null"
}

func newSessionID() string {
	bytesValue := make([]byte, 16)
	if _, err := rand.Read(bytesValue); err != nil {
		return ""
	}
	return hex.EncodeToString(bytesValue)
}
