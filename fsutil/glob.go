package fsutil

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// Glob return a filesystem that contain only files that match any of the provided
// patterns. If no patterns are provided, the original filesystem will be returned.
// An error will be returned if one of the patterns is invalid.
func Glob(fs http.FileSystem, patterns ...string) (http.FileSystem, error) {
	if len(patterns) == 0 {
		return fs, nil
	}
	if err := checkPatterns(patterns...); err != nil {
		return nil, err
	}
	return &glob{FileSystem: fs, patterns: patterns}, nil
}

// glob is an object that play the role of an http.FileSystem and an http.File.
// it wraps an existing underlying http.FileSystem, but applies glob pattern
// matching on its files.
type glob struct {
	http.FileSystem
	http.File
	root     string
	patterns []string
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
	if !g.match(path, info.IsDir()) {
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
		if g.match(path, file.IsDir()) {
			ret = append(ret, file)
		}
	}
	return ret, nil
}

// match a path to the defined patterns. If it is a file a full match
// is required. If it is a directory, only matching a prefix of any of
// the patterns is required.
func (g *glob) match(path string, isDir bool) bool {
	return (isDir && g.matchPrefix(path)) || (!isDir && g.matchFull(path))
}

// matchFull finds a matching of the whole name to any of the patterns.
func (g *glob) matchFull(name string) bool {
	for _, pattern := range g.patterns {
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
	}
	return false
}

// matchPrefix finds a matching of prefix to a prefix of any of the patterns.
func (g *glob) matchPrefix(prefix string) bool {
	parts := strings.Split(prefix, string(filepath.Separator))
nextPattern:
	for _, pattern := range g.patterns {
		patternParts := strings.Split(pattern, string(filepath.Separator))
		if len(patternParts) < len(parts) {
			continue
		}
		for i := 0; i < len(parts); i++ {
			if ok, _ := filepath.Match(patternParts[i], parts[i]); !ok {
				continue nextPattern
			}
		}
		return true
	}
	return false
}

// checkPattens checks the validity of the patterns.
func checkPatterns(patterns ...string) error {
	var badPatterns []string
	for _, pattern := range patterns {
		_, err := filepath.Match(pattern, "x")
		if err != nil {
			badPatterns = append(badPatterns, pattern)
			return errors.Wrap(err, pattern)
		}
	}
	if len(badPatterns) > 0 {
		return errors.Wrap(filepath.ErrBadPattern, strings.Join(badPatterns, ", "))
	}
	return nil
}
