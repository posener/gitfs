package tree

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/posener/gitfs/log"
)

// FS implements https://godoc.org/net/http#FileSystem over a Tree.
type FS struct {
	tree Tree
	ctx  context.Context
}

func NewFS(tree Tree) *FS {
	return &FS{tree: tree}
}

type Contexter interface {
	// Retruns a copy of the http.File with the given context for IO operations.
	WithContext(context.Context) http.File
}

func (fs FS) Open(name string) (http.File, error) {
	path := strings.Trim(name, "/")

	opener := fs.tree[path]
	if opener == nil {
		log.Printf("File %s not found", name)
		return nil, os.ErrNotExist
	}
	if !valid(name, opener.Stat) {
		log.Printf("File %s is invalid", name)
		return nil, os.ErrInvalid

	}

	f := opener.Open()

	// Apply context if available
	if fs.ctx != nil {
		if contexter, ok := f.(Contexter); ok {
			f = contexter.WithContext(fs.ctx)
		}
	}
	return f, nil
}

// WithContext returns a copy of filesystem with the given context.
// This context will be applied for open operations.
func (fs FS) WithContext(ctx context.Context) *FS {
	fs.ctx = ctx
	return &fs
}

func valid(name string, info func() (os.FileInfo, error)) bool {
	expectingDir := len(name) > 0 && name[len(name)-1] == '/'
	if expectingDir {
		if info, err := info(); err != nil || !info.IsDir() {
			return false
		}
	}
	return true
}
