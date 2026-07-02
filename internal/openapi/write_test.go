package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteWritesDocumentJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openapi.json")
	document := Generate([]Route{{Method: "GET", Path: "/api/tasks", OperationID: "listTasks", RequiresAuth: true}}, nil)

	result := Write(document, path)
	if _, written := result.(Written); !written {
		t.Fatalf("result = %#v, want Written", result)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("written file is empty")
	}
}

func TestWriteRejectsUnwritablePath(t *testing.T) {
	document := Generate(nil, nil)
	result := Write(document, filepath.Join(t.TempDir(), "missing-dir", "openapi.json"))
	if _, rejected := result.(WriteRejected); !rejected {
		t.Fatalf("result = %#v, want WriteRejected", result)
	}
}
