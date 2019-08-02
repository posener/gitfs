package binfs

import (
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

func TestLoadCalls(t *testing.T) {
	t.Parallel()
	got, err := LoadCalls("./testdata")
	require.NoError(t, err)

	want := Calls{
		project1: &Config{Project: project1, noPatterns: true},
		project2: &Config{Project: project2, globPatterns: []string{"foo", "*"}},
	}

	assert.Equal(t, want, got)
}

func TestLoadCalls_patternNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadCalls("./nosuchpackage")
	assert.Error(t, err)
}

func TestGenerateBinaries(t *testing.T) {
	var p testProvider

	calls := Calls{
		project1: &Config{Project: project1, noPatterns: true},
		project2: &Config{Project: project2, globPatterns: []string{"foo", "*"}},
	}

	binaries := GenerateBinaries(calls, p.provide)
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

type testProvider struct {
	// Saves with what projects the provider was called.
	calls []Config
}

func (p *testProvider) provide(c Config) (http.FileSystem, error) {
	p.calls = append(p.calls, c)
	return testFS(c.Project), nil
}

// testFS is a fake filesystem that contains only one file with
// the provided content.
func testFS(id string) http.FileSystem {
	t := make(tree.Tree)
	err := t.AddFileContent("dir/file", []byte(id))
	if err != nil {
		panic(err)
	}
	return t
}
