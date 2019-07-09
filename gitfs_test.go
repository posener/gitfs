package gitfs

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func ExampleTest() {
	ctx := context.Background()
	fs, err := New(ctx, "github.com/kelseyhightower/helloworld@tags/3.0.0")
	if err != nil {
		panic(err)
	}
	f, err := fs.Open("README.md")
	if err != nil {
		panic(err)
	}
	io.Copy(os.Stdout, f)
	// Output: # helloworld
}

func TestNew_notSupported(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := New(ctx, "git.com/nosuchusername/nosuchproject")
	require.Error(t, err)
}

func TestNew_local(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := New(ctx, "github.com/posener/gitfs", OptLocal("."))
	require.NoError(t, err)
}
