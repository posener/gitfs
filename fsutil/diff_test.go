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
	a.AddFileContent("content-equal", []byte("1\n2\n"))
	a.AddFileContent("common-dir/only-in-a", []byte(""))
	a.AddFileContent("file-in-a-dir-in-b", []byte(""))
	a.AddDir("dir-in-a-file-in-b")

	b := make(tree.Tree)
	b.AddFileContent("content-diff", []byte("1\n3\n"))
	b.AddFileContent("content-equal", []byte("1\n2\n"))
	b.AddFileContent("common-dir/only-in-b", []byte(""))
	b.AddDir("file-in-a-dir-in-b")
	b.AddFileContent("dir-in-a-file-in-b", []byte(""))

	want := `Diff between a and b:
[common-dir/only-in-a]: only in a
[common-dir/only-in-b]: only in b
[content-diff]: content diff (-a, +b):
 1
-2
+3
[dir-in-a-file-in-b]: on a is directory, on b file
[file-in-a-dir-in-b]: on a is file, on b directory
`
	got, err := Diff(a, b)
	require.NoError(t, err)
	assert.Equal(t, want, got.String())
}

func TestDiffEmpty(t *testing.T) {
	t.Parallel()

	a := make(tree.Tree)
	a.AddFileContent("foo", []byte(""))

	b := make(tree.Tree)

	got, err := Diff(a, b)
	require.NoError(t, err)
	assert.ElementsMatch(t, []PathDiff{{Path: "foo", Diff: msgOnlyInA}}, got.Diffs)

	// Mirror test
	got, err = Diff(b, a)
	require.NoError(t, err)
	assert.ElementsMatch(t, []PathDiff{{Path: "foo", Diff: msgOnlyInB}}, got.Diffs)
}
