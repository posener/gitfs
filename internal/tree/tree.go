package tree

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Opener interface {
	Open() http.File
	Stat() (os.FileInfo, error)
	Readdir(count int) ([]os.FileInfo, error)
}

type Tree map[string]Opener

type Loader func(context.Context) ([]byte, error)

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
