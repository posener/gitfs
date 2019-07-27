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

func TestNewWithGlob(t *testing.T) {
	tests := []struct {
		name     string
		prefetch bool
	}{
		{"no prefetch", false},
		{"prefetch", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := testFilesystem(t, "github.com/posener/gitfs/internal/testdata", tt.prefetch, []string{"*/*1"})
			require.NoError(t, err)
			_, err = fs.Open("d1/d11")
			assert.NoError(t, err)
			_, err = fs.Open("d1")
			assert.NoError(t, err)
			_, err = fs.Open("f01")
			assert.Error(t, err)
		})
	}
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

	_, err = ioutil.ReadAll(f21)
	require.Error(t, err)
}

func TestNewGithubProject(t *testing.T) {
	t.Parallel()
	p, err := newGithubFS(context.Background(), mockClient(), "github.com/x/y", nil)
	require.NoError(t, err)
	assert.Equal(t, "heads/master", p.ref)
}

func testFileSystemNoPrefetch(t *testing.T, project string) (http.FileSystem, error) {
	return testFilesystem(t, project, false, nil)
}

func testFileSystemPrefetch(t *testing.T, project string) (http.FileSystem, error) {
	return testFilesystem(t, project, true, nil)
}

func testFilesystem(t *testing.T, project string, prefetch bool, glob []string) (http.FileSystem, error) {
	t.Helper()
	if token == "" {
		t.Skip("no github token provided")
	}
	c := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
	return New(context.Background(), c, project, prefetch, glob)
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
