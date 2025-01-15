//go:build !ui
// +build !ui

package ui

import "io/fs"

// FS returns an empty filesystem when UI files are not embedded
func FS() fs.FS {
	return emptyFS{}
}

// emptyFS implements fs.FS and always returns fs.ErrNotExist
type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) {
	return nil, fs.ErrNotExist
}
