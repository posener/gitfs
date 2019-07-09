package tree

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/posener/gitfs/log"
)

func newFile(name string, size int64, load Loader) *file {
	return &file{name: name, size: size, load: load}
}

// file is an Opener for a file object.
type file struct {
	name string
	size int64
	load Loader

	content []byte
	mu      sync.Mutex
}

func (f *file) Open() http.File {
	return &lazyReader{file: f, ctx: context.Background()}
}

func (f *file) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *file) Name() string {
	return f.name
}

func (f *file) Size() int64 {
	return f.size
}

func (*file) Mode() os.FileMode {
	return 0
}

func (*file) ModTime() time.Time {
	return time.Time{}
}

func (*file) IsDir() bool {
	return false
}

func (*file) Sys() interface{} {
	return nil
}

func (*file) Readdir(count int) ([]os.FileInfo, error) {
	return nil, nil
}

func (f *file) loadContent(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.content != nil {
		return nil
	}
	start := time.Now()
	buf, err := f.load(ctx)
	if err != nil {
		return err
	}
	f.content = buf
	log.Printf("Loaded file %s in %.1fs", f.name, time.Now().Sub(start).Seconds())
	return nil
}

type lazyReader struct {
	*file
	reader *bytes.Reader
	ctx    context.Context
	mu     sync.Mutex
}

func (r *lazyReader) lazy() error {
	if err := r.loadContent(r.ctx); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.reader == nil {
		r.reader = bytes.NewReader(r.content)
	}
	return nil
}

func (r *lazyReader) WithContext(ctx context.Context) http.File {
	return r.withContext(ctx)
}

func (r lazyReader) withContext(ctx context.Context) *lazyReader {
	r.ctx = ctx
	return &r
}

func (r *lazyReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reader = nil
	r.ctx = context.Background()
	return nil
}

func (r *lazyReader) Read(p []byte) (int, error) {
	if err := r.lazy(); err != nil {
		return 0, err
	}
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func (r *lazyReader) Seek(offset int64, whence int) (int64, error) {
	if err := r.lazy(); err != nil {
		return 0, err
	}
	return r.reader.Seek(offset, whence)
}
