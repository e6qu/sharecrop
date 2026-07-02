package openapi

import (
	"encoding/json"
	"os"
)

type WriteResult interface {
	writeResult()
}

type Written struct{}

type WriteRejected struct {
	Reason string
}

func (Written) writeResult() {}

func (WriteRejected) writeResult() {}

func Write(document Document, path string) WriteResult {
	encoded, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return WriteRejected{Reason: "encode openapi document failed"}
	}
	if err := os.WriteFile(path, append(encoded, '\n'), 0o644); err != nil {
		return WriteRejected{Reason: "write openapi document failed"}
	}
	return Written{}
}
