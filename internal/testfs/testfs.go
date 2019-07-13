package testfs

import (
	"bytes"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFS(t *testing.T, fsFactory func(*testing.T, string) (http.FileSystem, error)) {
	tests := []struct {
		project string
		root    string
	}{
		{
			project: "github.com/posener/gitfs",
			root:    "internal/testdata",
		},
		{
			project: "github.com/posener/gitfs/internal",
			root:    "testdata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			fs, err := fsFactory(t, tt.project)
			require.NoError(t, err)
			fst := fsTest{
				FileSystem: fs,
				root:       tt.root,
			}
			t.Run("DirContains", fst.dirContains)
			t.Run("DirNotContains", fst.dirNotContains)
			t.Run("FileContent", fst.fileContent)
			t.Run("NotExistingFile", fst.notExistingFile)
		})
	}

	t.Run("NotSuchProject", func(t *testing.T) {
		_, err := fsFactory(t, "git.com/posener/gitfs")
		assert.Error(t, err)
	})
}

type fsTest struct {
	http.FileSystem
	root string
}

func (fs *fsTest) dirContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path     string
		contains string
		isDir    bool
	}{
		{path: fs.root, contains: "d1", isDir: true},
		{path: fs.root, contains: "d2", isDir: true},
		{path: fs.root, contains: "f01"},
		{path: fs.root + "/d1", contains: "d11", isDir: true},
		{path: fs.root + "/d1/d11", contains: "f111"},
		{path: fs.root + "/d2", contains: "f21"},
	}

	for _, tt := range tests {
		t.Run(tt.path+":"+tt.contains, func(t *testing.T) {
			f, err := fs.Open(tt.path)
			require.NoError(t, err)
			info := requireContains(t, f, tt.contains)
			assert.Equal(t, tt.isDir, info.IsDir())
		})
	}
}

func (fs *fsTest) dirNotContains(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path        string
		notContains string
	}{
		{path: fs.root, notContains: "d11"},
		{path: fs.root, notContains: "f111"},
		{path: fs.root, notContains: "d1/d11"},
	}

	for _, tt := range tests {
		t.Run(tt.path+":"+tt.notContains, func(t *testing.T) {
			f, err := fs.Open(tt.path)
			require.NoError(t, err)
			assertNotContains(t, f, tt.notContains)
		})
	}
}

func (fs *fsTest) fileContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path    string
		content string
	}{
		{path: fs.root + "/f01", content: ""},
		{path: fs.root + "/d1/d11/f111", content: "f111 content"},
		{path: fs.root + "/d2/f21", content: "f21 content"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			f, err := fs.Open(tt.path)
			require.NoError(t, err)
			assertFileContent(t, f, []byte(tt.content))
		})
	}
}

func (fs *fsTest) notExistingFile(t *testing.T) {
	t.Parallel()
	_, err := fs.Open(fs.root + "/nosuchfile")
	assert.Error(t, err)
}

func requireContains(t *testing.T, d http.File, contains string) os.FileInfo {
	t.Helper()
	files, err := d.Readdir(-1)
	require.NoError(t, err)
	for _, f := range files {
		if f.Name() == contains {
			return f
		}
	}
	t.Fatalf("FS did not contain file %q", contains)
	return nil
}

func assertNotContains(t *testing.T, d http.File, notContains string) {
	t.Helper()
	files, err := d.Readdir(-1)
	require.NoError(t, err)
	for _, f := range files {
		if f.Name() == notContains {
			t.Errorf("FS contains file %q", notContains)
			return
		}
	}
	return
}

func assertFileContent(t *testing.T, f http.File, content []byte) {
	t.Helper()
	gotContent := bytes.NewBuffer(nil)
	n, err := gotContent.ReadFrom(f)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), n)
	assert.Equal(t, content, gotContent.Bytes())
}
