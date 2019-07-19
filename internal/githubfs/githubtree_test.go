package githubfs

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/posener/gitfs/internal/testfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

var token = os.Getenv("GITHUB_TOKEN")

func TestNew(t *testing.T) {
	t.Run("NoPrefetch", func(t *testing.T) { testfs.TestFS(t, testFileSystemNoPrefetch) })
	t.Run("Prefetch", func(t *testing.T) { testfs.TestFS(t, testFileSystemPrefetch) })
}

type contexter interface {
	WithContext(context.Context) http.File
}

func TestOpen_cancelledContext(t *testing.T) {
	t.Parallel()
	fs, err := testFileSystemNoPrefetch(t, "github.com/posener/gitfs")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f21, err := fs.Open("internal/testdata/f01")
	require.NoError(t, err)

	f21Ctx, ok := f21.(contexter)
	require.True(t, ok)
	f21 = f21Ctx.WithContext(ctx)

	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(f21)
	require.Error(t, err)
}

func TestGithubProjectProperties(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path      string
		wantOwner string
		wantRepo  string
		wantRef   string
		wantPath  string
	}{
		{path: "github.com/x/y@tags/v1", wantOwner: "x", wantRepo: "y", wantRef: "tags/v1"},
		{path: "github.com/x/y@heads/foo", wantOwner: "x", wantRepo: "y", wantRef: "heads/foo"},
		{path: "github.com/x/y", wantOwner: "x", wantRepo: "y", wantRef: ""},
		{path: "github.com/x/y@v1", wantOwner: "x", wantRepo: "y", wantRef: "tags/v1"},
		{path: "github.com/x/y@v1.2", wantOwner: "x", wantRepo: "y", wantRef: "tags/v1.2"},
		{path: "github.com/x/y@v1.2.3", wantOwner: "x", wantRepo: "y", wantRef: "tags/v1.2.3"},
		{path: "github.com/x/y@1", wantOwner: "x", wantRepo: "y", wantRef: "tags/1"},
		{path: "github.com/x/y@1.2", wantOwner: "x", wantRepo: "y", wantRef: "tags/1.2"},
		{path: "github.com/x/y@1.2.3", wantOwner: "x", wantRepo: "y", wantRef: "tags/1.2.3"},
		{path: "github.com/x/y/static/path", wantOwner: "x", wantRepo: "y", wantPath: "static/path/"},
		{path: "github.com/x/y/static@v1.2.3", wantOwner: "x", wantRepo: "y", wantRef: "tags/v1.2.3", wantPath: "static/"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			gotOwner, gotRepo, gotPath, gotRef, err := githubProjectProperties(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantOwner, gotOwner)
			assert.Equal(t, tt.wantRepo, gotRepo)
			assert.Equal(t, tt.wantRef, gotRef)
			assert.Equal(t, tt.wantPath, gotPath)
		})
	}
}

func TestGithubProjectProperties_error(t *testing.T) {
	t.Parallel()
	paths := []string{
		// Not github.com
		"google.com/x/y@tags/v1",
		// Not .com
		"github/x/y@tags/v1",
		// Missing repo
		"github.com/x@tags/v1",
		// Missing owner and repo
		"github.com@tags/v1",
		// Invalid reference
		"github.com/x/y@x1",
		// Invalid semvers
		"github.com/x/y@v1.",
		"github.com/x/y@v1.2.3.4",
		"github.com/x/y@1.",
		"github.com/x/y@1.2.3.4",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			owner, repo, path, ref, err := githubProjectProperties(path)
			assert.Error(t, err, "Got owner=%s repo=%s path=%s ref=%s", owner, repo, path, ref)
		})
	}
}

func TestNewGithubProject(t *testing.T) {
	t.Parallel()
	p, err := newGithubProject(context.Background(), mockClient(), "github.com/x/y")
	require.NoError(t, err)
	assert.Equal(t, "heads/master", p.ref)
}

func testFileSystemNoPrefetch(t *testing.T, project string) (http.FileSystem, error) {
	return testFilesystem(t, project, false)
}

func testFileSystemPrefetch(t *testing.T, project string) (http.FileSystem, error) {
	return testFilesystem(t, project, true)
}

func testFilesystem(t *testing.T, project string, prefetch bool) (http.FileSystem, error) {
	t.Helper()
	if token == "" {
		t.Skip("no github token provided")
	}
	c := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	return New(context.Background(), c, project, prefetch)
}

func mockClient() *http.Client {
	return &http.Client{Transport: &mockTransport{}}
}

type mockTransport struct{}

func (*mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case req.Method == http.MethodGet && req.URL.Path == "/repos/x/y":
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"default_branch":"master"}`))),
			Request:    req,
		}, nil
	default:
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Header:     make(http.Header),
			Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
			Request:    req,
		}, nil
	}
}
