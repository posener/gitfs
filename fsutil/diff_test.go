package fsutil

import (
	"testing"

	"github.com/posener/gitfs/internal/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiff(t *testing.T) {
	t.Parallel()

	a := make(tree.Tree)
	a.AddFileContent("content-diff", []byte("1\n2\n"))
	a.AddFileContent("common-dir/only-in-a", []byte(""))
	a.AddFileContent("file-in-a-dir-in-b", []byte(""))
	a.AddDir("dir-in-a-file-in-b")

	b := make(tree.Tree)
	b.AddFileContent("content-diff", []byte("1\n3\n"))
	b.AddFileContent("common-dir/only-in-b", []byte(""))
	b.AddDir("file-in-a-dir-in-b")
	b.AddFileContent("dir-in-a-file-in-b", []byte(""))

	want := FileSystemDiff{
		{Path: "common-dir/only-in-a", Diff: msgOnlyInA},
		{Path: "common-dir/only-in-b", Diff: msgOnlyInB},
		{Path: "file-in-a-dir-in-b", Diff: msgAFileBDir},
		{Path: "dir-in-a-file-in-b", Diff: msgADirBFile},
		{Path: "content-diff", Diff: "content diff:\n 1\n-2\n+3"},
	}

	got, err := Diff(a, b)
	require.NoError(t, err)
	assert.ElementsMatch(t, want, got)
}

func TestDiffEmpty(t *testing.T) {
	t.Parallel()

	a := make(tree.Tree)
	a.AddFileContent("foo", []byte(""))

	b := make(tree.Tree)

	got, err := Diff(a, b)
	require.NoError(t, err)
	assert.ElementsMatch(t, FileSystemDiff{{Path: "foo", Diff: msgOnlyInA}}, got)

	// Mirror test
	got, err = Diff(b, a)
	require.NoError(t, err)
	assert.ElementsMatch(t, FileSystemDiff{{Path: "foo", Diff: msgOnlyInB}}, got)
}
