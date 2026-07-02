package openapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func parseFiles(t *testing.T, sources map[string]string) map[string]*ast.File {
	t.Helper()
	fset := token.NewFileSet()
	files := map[string]*ast.File{}
	for name, src := range sources {
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files[name] = file
	}
	return files
}

func TestResolveDTOTypesFindsInlineDecodeAndCompositeLiteralResponse(t *testing.T) {
	files := parseFiles(t, map[string]string{"x.go": `package httpserver

import (
	"encoding/json"
	"net/http"
)

func (server Server) createTask(w http.ResponseWriter, r *http.Request) {
	var request taskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid")
		return
	}
	response := taskResponse{Title: request.Title}
	writeJSON(w, http.StatusCreated, response)
}

func writeError(w http.ResponseWriter, status int, message string) {}
func writeJSON(w http.ResponseWriter, status int, value writableResponse) {}
type writableResponse interface{ writableResponse() }
`})

	requestTypes, responseTypes := resolveDTOTypes(files)
	if requestTypes["createTask"] != "taskRequest" {
		t.Fatalf("request type = %q, want taskRequest", requestTypes["createTask"])
	}
	if responseTypes["createTask"] != "taskResponse" {
		t.Fatalf("response type = %q, want taskResponse", responseTypes["createTask"])
	}
}

func TestResolveDTOTypesFollowsDedicatedWriteWrapper(t *testing.T) {
	files := parseFiles(t, map[string]string{"x.go": `package httpserver

import "net/http"

func (server Server) login(w http.ResponseWriter, r *http.Request) {
	server.writeAuthResponse(w, http.StatusOK, authResponse{})
}

func (server Server) writeAuthResponse(w http.ResponseWriter, status int, response authResponse) {}
`})

	_, responseTypes := resolveDTOTypes(files)
	if responseTypes["login"] != "authResponse" {
		t.Fatalf("response type = %q, want authResponse (via method-selector wrapper call)", responseTypes["login"])
	}
}

func TestResolveDTOTypesFollowsConverterFunctionReturnType(t *testing.T) {
	files := parseFiles(t, map[string]string{"x.go": `package httpserver

import "net/http"

func (server Server) mintCollectible(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusCreated, collectibleToResponse(0))
}

func collectibleToResponse(value int) collectibleResponse { return collectibleResponse{} }
func writeJSON(w http.ResponseWriter, status int, value writableResponse) {}
type writableResponse interface{ writableResponse() }
`})

	_, responseTypes := resolveDTOTypes(files)
	if responseTypes["mintCollectible"] != "collectibleResponse" {
		t.Fatalf("response type = %q, want collectibleResponse", responseTypes["mintCollectible"])
	}
}

func TestResolveDTOTypesIsTransitiveThroughSharedHelper(t *testing.T) {
	files := parseFiles(t, map[string]string{"x.go": `package httpserver

import "net/http"

func (server Server) openTask(w http.ResponseWriter, r *http.Request) {
	server.changeTaskState(w, r)
}

func (server Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	server.changeTaskState(w, r)
}

func (server Server) changeTaskState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, taskResponse{})
}

func writeJSON(w http.ResponseWriter, status int, value writableResponse) {}
type writableResponse interface{ writableResponse() }
`})

	_, responseTypes := resolveDTOTypes(files)
	if responseTypes["openTask"] != "taskResponse" {
		t.Fatalf("openTask response type = %q, want taskResponse via changeTaskState", responseTypes["openTask"])
	}
	if responseTypes["cancelTask"] != "taskResponse" {
		t.Fatalf("cancelTask response type = %q, want taskResponse via changeTaskState", responseTypes["cancelTask"])
	}
}

// TestResolveDTOTypesIgnoresGenericErrorHelper locks in a real bug: writeError
// has the same (http.ResponseWriter, int, <named type>) shape as a real
// write<Foo>Response wrapper, so an early call to it (on an error path
// checked before the real success-path write) must not be mistaken for the
// handler's response type.
func TestResolveDTOTypesIgnoresGenericErrorHelper(t *testing.T) {
	files := parseFiles(t, map[string]string{"x.go": `package httpserver

import "net/http"

func (server Server) createTask(w http.ResponseWriter, r *http.Request) {
	if !server.ok(r) {
		writeError(w, http.StatusUnauthorized, "nope")
		return
	}
	writeJSON(w, http.StatusCreated, taskResponse{})
}

func writeError(w http.ResponseWriter, status int, message string) {}
func writeJSON(w http.ResponseWriter, status int, value writableResponse) {}
type writableResponse interface{ writableResponse() }
`})

	_, responseTypes := resolveDTOTypes(files)
	if responseTypes["createTask"] != "taskResponse" {
		t.Fatalf("response type = %q, want taskResponse (writeError must not be treated as the response wrapper)", responseTypes["createTask"])
	}
}

// TestResolveDTOTypesIgnoresWriteJSONsOwnParameterType locks in a real bug:
// writeJSON's own third parameter is the writableResponse marker interface,
// which must not be recorded as a "wrapper" response type for writeJSON
// itself, or every writeJSON call site would resolve to "writableResponse"
// instead of the actual argument's type.
func TestResolveDTOTypesIgnoresWriteJSONsOwnParameterType(t *testing.T) {
	files := parseFiles(t, map[string]string{"x.go": `package httpserver

import "net/http"

func (server Server) listCollectibles(w http.ResponseWriter, r *http.Request) {
	response := collectiblesResponse{}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, value writableResponse) {}
type writableResponse interface{ writableResponse() }
`})

	_, responseTypes := resolveDTOTypes(files)
	if responseTypes["listCollectibles"] != "collectiblesResponse" {
		t.Fatalf("response type = %q, want collectiblesResponse, not writableResponse", responseTypes["listCollectibles"])
	}
}

func TestResolveDTOTypesLeavesUnmatchedHandlersUnresolved(t *testing.T) {
	files := parseFiles(t, map[string]string{"x.go": `package httpserver

import "net/http"

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
`})

	requestTypes, responseTypes := resolveDTOTypes(files)
	if _, ok := requestTypes["health"]; ok {
		t.Fatalf("health must have no resolved request type, got %q", requestTypes["health"])
	}
	if _, ok := responseTypes["health"]; ok {
		t.Fatalf("health must have no resolved response type, got %q", responseTypes["health"])
	}
}
