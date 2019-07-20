// Package fsutil provides useful utility functions for http.FileSystem.
package fsutil

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/kr/fs"
)

// Walk returns a fs.Walker over http.FileSystem, which enables
// walking over all files in the filesystem.
//
// See https://godoc.org/github.com/kr/fs#Walker for more details.
func Walk(hfs http.FileSystem, root string) *fs.Walker {
	return fs.WalkFS(root, fileSystem{hfs})
}

// FileSystem implements fs.FileSystem over http.FileSystem.
//
// See https://godoc.org/github.com/kr/fs#FileSystem for more details.
type fileSystem struct {
	http.FileSystem
}

func (fs fileSystem) ReadDir(dirname string) ([]os.FileInfo, error) {
	f, err := fs.Open(dirname)
	if err != nil {
		return nil, err
	}
	return f.Readdir(-1)
}

func (fs fileSystem) Lstat(name string) (os.FileInfo, error) {
	f, err := fs.Open(name)
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

func (fileSystem) Join(elem ...string) string {
	return filepath.Join(elem...)
}
