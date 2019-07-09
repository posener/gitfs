package tree

import (
	"net/http"
	"os"
	"time"
)

func newDir(name string) *dir {
	return &dir{name: name}
}

type dir struct {
	name  string
	files []os.FileInfo
}

func (d *dir) Open() http.File {
	return d
}

func (d *dir) add(f os.FileInfo) {
	d.files = append(d.files, f)
}

func (d *dir) Close() error {
	return nil
}
func (d *dir) Read([]byte) (int, error) {
	return 0, nil
}
func (d *dir) Seek(int64, int) (int64, error) {
	return 0, nil
}

func (d *dir) Readdir(n int) ([]os.FileInfo, error) {
	if n <= 0 || n >= len(d.files) {
		return d.files, nil
	}
	return d.files[:n], nil
}

func (d *dir) Stat() (os.FileInfo, error) {
	return d, nil
}

func (d *dir) Name() string {
	return d.name
}
func (d *dir) Size() int64 {
	return 0
}
func (d *dir) Mode() os.FileMode {
	return os.ModeDir
}
func (d *dir) ModTime() time.Time {
	return time.Time{}
}

func (d *dir) IsDir() bool {
	return true
}

func (d *dir) Sys() interface{} {
	return nil
}
