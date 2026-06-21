package web

import (
	"embed"
	"io/fs"
)

//go:embed static/*
var static embed.FS

func StaticFiles() (fs.FS, error) {
	return fs.Sub(static, "static")
}
