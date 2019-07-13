package tree

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Opener is an interface for a directory or a file provider.
type Opener interface {
	// Open returns a file/dir.
	Open() http.File
	// Stat returns information about the file/dir.
	Stat() (os.FileInfo, error)
	// Readdir retruns list of file info contained in a directory.
	// Preforming Readdir on a file returns nil, nil.
	Readdir(count int) ([]os.FileInfo, error)
}

// Tree maps a file path to a file provider.
type Tree map[string]Opener

// Loader is a function that loads file content. If the context id done
// this function should return an error.
type Loader func(context.Context) ([]byte, error)

// AddDir adds a directory to a tree. It also adds recursively all the
// parent directories.
func (t Tree) AddDir(path string) error {
	path = cleanPath(path)
	if t[path] != nil {
		if _, ok := t[path].(*dir); !ok {
			return fmt.Errorf("trying to override %T on path %s with a dir", t[path], path)
		}
		return nil
	}
	dirPath, name := filepath.Split(path)
	dirPath = cleanPath(dirPath)
	d := newDir(name)
	t[path] = d

	// Skip setting parent directory for root directory.
	if name == "" {
		return nil
	}

	// Add parent directory, and add the current directory to the parent.
	err := t.AddDir(dirPath)
	if err != nil {
		return err
	}
	st, _ := d.Stat()
	parent, ok := t[dirPath].(*dir)
	if !ok {
		panic(fmt.Sprintf("Expected %q to be *dir, got %T", dirPath, t[dirPath]))
	}
	parent.add(st)
	return nil
}

// AddFile adds a file to a tree. It also adds recursively all the
// parent directories.
func (t Tree) AddFile(path string, size int, load Loader) error {
	path = cleanPath(path)
	if t[path] != nil {
		if _, ok := t[path].(*file); !ok {
			return fmt.Errorf("trying to override %T on path %s with a file", t[path], path)
		}
		return nil
	}
	dirPath, name := filepath.Split(path)
	dirPath = cleanPath(dirPath)
	f := newFile(name, int64(size), load)
	t[path] = f

	// Add parent directory, and add the current file to the parent.
	err := t.AddDir(dirPath)
	if err != nil {
		return err
	}
	parent, ok := t[dirPath].(*dir)
	if !ok {
		panic(fmt.Sprintf("Expected %q to be *dir, got %T", dirPath, t[dirPath]))
	}
	parent.add(f)
	return nil
}

func cleanPath(path string) string {
	return strings.Trim(path, "/")
}
