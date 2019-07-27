package fsutil

import (
	"net/http"
	"os"
	"path/filepath"

	globutil "github.com/posener/gitfs/internal/glob"
)

// Glob return a filesystem that contain only files that match any of the provided
// patterns. If no patterns are provided, the original filesystem will be returned.
// An error will be returned if one of the patterns is invalid.
func Glob(fs http.FileSystem, patterns ...string) (http.FileSystem, error) {
	if len(patterns) == 0 {
		return fs, nil
	}
	p, err := globutil.New(patterns...)
	if err != nil {
		return nil, err
	}
	return &glob{FileSystem: fs, patterns: p}, nil
}

// glob is an object that play the role of an http.FileSystem and an http.File.
// it wraps an existing underlying http.FileSystem, but applies glob pattern
// matching on its files.
type glob struct {
	http.FileSystem
	http.File
	root     string
	patterns globutil.Patterns
}

// Open a file, relative to root. If the file exists in the filesystem
// but does not match any of the patterns an os.ErrNotExist will be
// returned. If name is a directory, but it does not match the prefix
// of any of the patterns, and os.ErrNotExist will be returned.
func (g *glob) Open(name string) (http.File, error) {
	path := filepath.Join(g.root, name)
	f, err := g.FileSystem.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	// Regular file, match name.
	if !g.patterns.Match(path, info.IsDir()) {
		return nil, os.ErrNotExist
	}
	return &glob{
		FileSystem: g.FileSystem,
		File:       f,
		root:       path,
		patterns:   g.patterns,
	}, nil
}

// Readdir returns a list of files that match the patterns.
func (g *glob) Readdir(count int) ([]os.FileInfo, error) {
	files, err := g.File.Readdir(count)
	if err != nil {
		return nil, err
	}
	ret := make([]os.FileInfo, 0, len(files))
	for _, file := range files {
		path := filepath.Join(g.root, file.Name())
		if g.patterns.Match(path, file.IsDir()) {
			ret = append(ret, file)
		}
	}
	return ret, nil
}
