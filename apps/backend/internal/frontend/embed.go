package frontend

import (
	"embed"
	"io/fs"
)

//go:embed web/**
var Frontend embed.FS

func Files() (fs.FS, error) {
	return fs.Sub(Frontend, "web")
}
