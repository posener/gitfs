package tree

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTree(t *testing.T) {
	t.Parallel()

	tr := make(Tree)
	require.NoError(t, tr.AddDir("a"))
	assertDir(t, tr, "")
	assertDir(t, tr, "a")
	assertDirContains(t, tr, "", "a")

	require.NoError(t, tr.AddDir("a/b"))
	assertDir(t, tr, "a")
	assertDir(t, tr, "a/b")
	assertDirContains(t, tr, "a", "b")

	require.NoError(t, tr.AddDir("a/bb"))
	assertDir(t, tr, "a")
	assertDir(t, tr, "a/b")
	assertDir(t, tr, "a/bb")
	assertDirContains(t, tr, "a", "b")
	assertDirContains(t, tr, "a", "bb")

	require.NoError(t, tr.AddDir("c/d"))
	assertDir(t, tr, "c")
	assertDir(t, tr, "c/d")
	assertDirContains(t, tr, "", "c")
	assertDirContains(t, tr, "c", "d")

	require.NoError(t, tr.AddFile("a/f1", 10, nil))
	assertFile(t, tr, "a/f1", 10)
	assertDirContains(t, tr, "a", "f1")

	require.NoError(t, tr.AddFile("e/f1", 10, nil))
	assertFile(t, tr, "e/f1", 10)
	assertDir(t, tr, "e")
	assertDirContains(t, tr, "e", "f1")
}

func TestDir_readDir(t *testing.T) {
	t.Parallel()

	tr := make(Tree)
	require.NoError(t, tr.AddFile("a/1", 0, nil))
	require.NoError(t, tr.AddFile("a/2", 0, nil))
	require.NoError(t, tr.AddFile("a/3", 0, nil))

	tests := []struct {
		count   int
		wantLen int
	}{
		{count: -1, wantLen: 3},
		{count: 0, wantLen: 3},
		{count: 1, wantLen: 1},
		{count: 2, wantLen: 2},
		{count: 3, wantLen: 3},
		{count: 4, wantLen: 3},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.count), func(t *testing.T) {
			files, err := tr["a"].Readdir(tt.count)
			require.NoError(t, err)
			assert.Len(t, files, tt.wantLen)
		})
	}
}

func TestFile_read(t *testing.T) {
	t.Parallel()

	content := "content"

	tr := make(Tree)
	require.NoError(t, tr.AddFile("a", len(content), contentProvider(content)))

	assertContent(t, tr["a"].Open(), content)
}

func TestFile_readFailure(t *testing.T) {
	t.Parallel()

	tr := make(Tree)
	require.NoError(t, tr.AddFile("a", 10, func(context.Context) ([]byte, error) { return nil, fmt.Errorf("failed") }))
	assert.NotNil(t, tr["a"])

	buf := make([]byte, 10)
	_, err := tr["a"].Open().Read(buf)
	assert.Error(t, err)
}

func TestFile_overrideFailure(t *testing.T) {
	t.Parallel()

	tr := make(Tree)
	assert.NoError(t, tr.AddFile("a", 10, nil))
	assert.Error(t, tr.AddDir("a"))

	assert.NoError(t, tr.AddDir("b"))
	assert.Error(t, tr.AddFile("b", 10, nil))
}

func assertDir(t *testing.T, tr Tree, path string) {
	t.Helper()
	require.NotNil(t, tr[path])
	st, err := tr[path].Stat()
	require.NoError(t, err)
	assert.True(t, st.IsDir())
}

func assertDirContains(t *testing.T, tr Tree, path string, contains string) {
	t.Helper()
	require.NotNil(t, tr[path])
	files, err := tr[path].Readdir(-1)
	require.NoError(t, err)
	for _, f := range files {
		if f.Name() == contains {
			return
		}
	}
	t.Errorf("Dir %q did not contain file %q", path, contains)
}

func assertFile(t *testing.T, tr Tree, path string, size int64) {
	t.Helper()
	require.NotNil(t, tr[path])
	st, err := tr[path].Stat()
	require.NoError(t, err)
	assert.False(t, st.IsDir())
	assert.Equal(t, st.Size(), size)
}

func assertContent(t *testing.T, r io.Reader, content string) {
	t.Helper()
	require.NotNil(t, r)
	gotContent := bytes.NewBuffer(nil)
	n, err := gotContent.ReadFrom(r)
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), n)
	assert.Equal(t, content, gotContent.String())
}

func contentProvider(content string) func(context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return []byte(content), nil
	}
}
