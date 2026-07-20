package web

import (
	"bytes"
	"embed"
	"io/fs"
)

//go:embed static/*
var static embed.FS

func StaticFiles() (fs.FS, error) {
	return fs.Sub(static, "static")
}

const shauthFlag = "shauth: false, // SHARECROP_SHAUTH_FLAG"

func ApplicationShell(files fs.FS, shauthEnabled bool) ([]byte, error) {
	data, err := fs.ReadFile(files, "index.html")
	if err != nil {
		return nil, err
	}
	if !shauthEnabled {
		return data, nil
	}
	return bytes.ReplaceAll(data, []byte(shauthFlag), []byte("shauth: true, // SHARECROP_SHAUTH_FLAG")), nil
}
