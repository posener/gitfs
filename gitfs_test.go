package gitfs

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/posener/gitfs/fsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// With gitfs you can open a remote git repository, and load any file,
// including non-go files.
// In this example, the README.md file of a remote repository is loaded.
func Example_open() {
	ctx := context.Background()

	// The load path is of the form: github.com/<owner>/<repo>(/<path>)?(@<ref>)?.
	// `ref` can reference any git tag or branch. If github releases are in Semver format,
	// the `tags/` prefix is not needed in the `ref` part.
	fs, err := New(ctx, "github.com/kelseyhightower/helloworld@3.0.0")
	if err != nil {
		log.Fatalf("Failed initialize filesystem: %s", err)
	}

	// Open any file in the github repository, using the `Open` function. Both files
	// and directory can be opened. The content is not loaded until it is actually being
	// read. The content is loaded only once.
	f, err := fs.Open("README.md")
	if err != nil {
		log.Fatalf("Failed opening file: %s", err)
	}

	// Copy the content to stdout.
	io.Copy(os.Stdout, f)

	// Output: # helloworld
}

// The ./fsutil package is a collection of useful functions that can work with
// any `http.FileSystem` implementation.
// For example, here we will use a function that loads go templates from the
// filesystem.
func Example_fsutil() {
	ctx := context.Background()

	// Open a git remote repository `posener/gitfs` in path `examples/templates`.
	fs, err := New(ctx, "github.com/posener/gitfs/examples/templates")
	if err != nil {
		log.Fatalf("Failed initialize filesystem: %s", err)
	}

	// Use util function that loads all templates according to a glob pattern.
	tmpls, err := fsutil.TmplParseGlob(fs, nil, "*.gotmpl")
	if err != nil {
		log.Fatalf("Failed parsing templates: %s", err)
	}

	// Execute the template and write to stdout.
	tmpls.ExecuteTemplate(os.Stdout, "tmpl1.gotmpl", "Foo")

	// Output: Hello, Foo
}

// Tests not supported repository pattern.
func TestNew_notSupported(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := New(ctx, "git.com/nosuchusername/nosuchproject")
	require.Error(t, err)
}

// Tests loading of local repository.
func TestNew_local(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	_, err := New(ctx, "github.com/posener/gitfs", OptLocal("."))
	require.NoError(t, err)
}

func TestWithContext(t *testing.T) {
	t.Parallel()
	fs, err := New(context.Background(), "github.com/posener/gitfs")
	require.NoError(t, err)
	f, err := fs.Open("README.md")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f = WithContext(f, ctx)
	_, err = f.Read(make([]byte, 10))
	assert.EqualError(t, err, "failed getting blob: context canceled")
}

func init() {
	// Set Github access token in default client if available
	// from environment variables.
	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		http.DefaultClient = oauth2.NewClient(
			context.Background(),
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	}
}
