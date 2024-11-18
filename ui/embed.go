package ui

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var ui embed.FS

// FS returns the embedded distribution of the Glu UI.
func FS() fs.FS {
	fs, err := fs.Sub(ui, "dist")
	if err != nil {
		panic(err)
	}

	return fs
}
