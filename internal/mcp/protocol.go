package mcp

import "encoding/json"

const (
	jsonRPCVersion  = "2.0"
	protocolVersion = "2025-06-18"
	serverName      = "sharecrop"
	serverVersion   = "0.1.0"
)

func ProtocolVersion() string {
	return protocolVersion
}

const (
	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeInternalError  = -32603
	codeScopeDenied    = -32001
)

// Request is a single JSON-RPC 2.0 request or notification.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// Response is a single JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func successResponse(id json.RawMessage, result json.RawMessage) Response {
	return Response{JSONRPC: jsonRPCVersion, ID: id, Result: result}
}

func errorResponse(id json.RawMessage, code int, message string) Response {
	return Response{JSONRPC: jsonRPCVersion, ID: id, Error: &RPCError{Code: code, Message: message}}
}
