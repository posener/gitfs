package tree

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen(t *testing.T) {
	t.Parallel()
	tr := make(Tree)
	require.NoError(t, tr.AddFile("a", 6, contentProvider("file a")))
	require.NoError(t, tr.AddFile("b/c", 6, contentProvider("file c")))
	fs := NewFS(tr)

	a, err := fs.Open("a")
	require.NoError(t, err)
	assertContent(t, a, "file a")

	a, err = fs.Open("/a")
	require.NoError(t, err)
	assertContent(t, a, "file a")

	// Not found
	_, err = fs.Open("nosuchfile")
	assert.EqualError(t, err, os.ErrNotExist.Error())

	// Invalid - a is a file not a directory.
	_, err = fs.Open("a/")
	assert.EqualError(t, err, os.ErrInvalid.Error())

	// Test cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a, err = fs.WithContext(ctx).Open("a")
	require.NoError(t, err)
	_, err = a.Read(make([]byte, 10))
	assert.EqualError(t, err, context.Canceled.Error())
}
