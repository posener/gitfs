package binfs

import (
	"context"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/posener/gitfs/internal/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	project1 = "github.com/a/b"
	project2 = "github.com/c/d"
)

func TestLoadSimulation(t *testing.T) {
	var p testProvider

	// Simulate a `gitfs ./testdata` command.
	// Instead of running the gitfs command which invokes `LoadBinaries` and
	// creates a go file with `Register()` calls, we invoke loadBinaries with
	// the same arguments and call the `Register()` function with the result
	// instead.
	binaries, err := LoadBinaries([]string{"./testdata"}, p.provide)
	require.NoError(t, err)

	// Check that two calls to `gitfs`.New with the right projects
	// were observed, as this is what ./testdata/testdata.go calls.
	assert.ElementsMatch(t, []call{{project1, nil}, {project2, []string{"foo", "*"}}}, p.calls)

	// Register the data that was created by loadBinaries.
	for _, project := range []string{project1, project2} {
		data := binaries[project]
		require.NotNil(t, data)
		Register(project, EncodeVersion, data)
	}

	// Check the data that was registered:
	for _, project := range []string{project1, project2} {
		assert.True(t, Match(project))
		fs := Get(project)
		require.NotNil(t, fs)
		f, err := fs.Open("dir/file")
		assert.NoError(t, err)
		b, err := ioutil.ReadAll(f)
		f.Close()
		assert.NoError(t, err)
		assert.Equal(t, project, string(b))
	}
}

func TestLoadBinaries_patternNotFound(t *testing.T) {
	t.Parallel()

	var p testProvider

	_, err := LoadBinaries([]string{"./nosuchpackage"}, p.provide)

	assert.Error(t, err)
	assert.Equal(t, 0, len(p.calls))
}

type testProvider struct {
	// Saves with what projects the provider was called.
	calls []call
}

type call struct {
	project  string
	patterns []string
}

func (p *testProvider) provide(project string, patterns []string) (http.FileSystem, error) {
	p.calls = append(p.calls, call{project, patterns})
	return testFS(project), nil
}

// testFS is a fake filesystem that contains only one file with
// the provided content.
func testFS(id string) http.FileSystem {
	t := make(tree.Tree)
	t.AddFile("dir/file", 10, func(context.Context) ([]byte, error) { return []byte(id), nil })
	return t
}
