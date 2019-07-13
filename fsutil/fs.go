// Package fsutil provides utility functions for http filesystems.
package fsutil

import (
	"net/http"
	"os"
	"path/filepath"
	"github.com/kr/fs"
)

// Walk returns a https://godoc.org/github.com/kr/fs#Walker over an
// https://golang.org/pkg/net/http/#FileSystem.
func Walk(hfs http.FileSystem, root string) *fs.Walker {
	return fs.WalkFS(root, fileSystem{hfs})
}

// FileSystem implements https://godoc.org/github.com/kr/fs#FileSystem over
// https://golang.org/pkg/net/http/#FileSystem. It can be used to walking
// over files in the filesystem.
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
