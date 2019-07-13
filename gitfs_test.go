package gitfs

import (
	"context"
	"io"
	"log"
	"os"
	"testing"

	"github.com/posener/gitfs/fsutil"
	"github.com/stretchr/testify/require"
)

func ExampleTest() {
	ctx := context.Background()
	fs, err := New(ctx, "github.com/kelseyhightower/helloworld@tags/3.0.0")
	if err != nil {
		log.Fatalf("Failed initialize filesystem: %s", err)
	}
	f, err := fs.Open("README.md")
	if err != nil {
		log.Fatalf("Failed opening file: %s", err)
	}
	io.Copy(os.Stdout, f)
	// Output: # helloworld
}

func ExampleTest_templates() {
	ctx := context.Background()
	fs, err := New(ctx, "github.com/posener/gitfs/examples/templates")
	if err != nil {
		log.Fatalf("Failed initialize filesystem: %s", err)
	}

	tmpls, err := fsutil.TmplParseGlob(fs, nil, "*.gotmpl")
	if err != nil {
		log.Fatalf("Failed parsing templates: %s", err)
	}
	tmpls.ExecuteTemplate(os.Stdout, "tmpl1.gotmpl", "Foo")
	// Output: Hello, Foo
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
