package contracts

import (
	"os"
	"path/filepath"
)

type WriteElmFilesResult interface {
	writeElmFilesResult()
}

type ElmFilesWritten struct{}

type WriteElmFilesRejected struct {
	Reason string
}

func (ElmFilesWritten) writeElmFilesResult() {}

func (WriteElmFilesRejected) writeElmFilesResult() {}

func WriteElmFiles(files []ElmFile) WriteElmFilesResult {
	for _, file := range files {
		if err := os.MkdirAll(filepath.Dir(file.Path), 0o755); err != nil {
			return WriteElmFilesRejected{Reason: "create generated Elm directory failed"}
		}
		if err := os.WriteFile(file.Path, []byte(file.Content), 0o644); err != nil {
			return WriteElmFilesRejected{Reason: "write generated Elm file failed"}
		}
	}
	return ElmFilesWritten{}
}
