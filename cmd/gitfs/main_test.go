package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	clean()
	t.Run("Successful run", func(t *testing.T) {
		t.Parallel()
		stderr, err := runGo(t, "run", ".", "-out", "testout1.go", "../../examples/templates/...")
		assert.NoErrorf(t, err, "Expected success, got error: %s", stderr)

		// Test the output file.
		data, err := ioutil.ReadFile("testout1.go")
		require.NoError(t, err)
		assert.True(t, regexp.MustCompile(`package main`).Match(data))

		stderr, err = runGo(t, "build", ".")
		require.NoErrorf(t, err, "Build failed: %s", stderr)
		stderr, err = runGo(t, "test", "./...", "-run", "TestGitFS")
		require.NoErrorf(t, err, "Test failed: %s", stderr)
	})

	t.Run("Pattern must be provided", func(t *testing.T) {
		t.Parallel()
		_, err := runGo(t, "run", ".", "-out", "testout2.go")
		assert.Error(t, err)

		// Test that file was deleted after failure.
		_, err = os.Stat("testout2.go")
		assert.Error(t, err)
	})
}

func TestGetOut(t *testing.T) {
	t.Parallel()
	tests := []struct {
		out  string
		want string
	}{
		{out: "", want: "gitfs.go"},
		// A local file stay the same.
		{out: "f.go", want: "f.go"},
		// A dir is appended with file.
		{out: "/tmp", want: "/tmp/gitfs.go"},
		// Local dir is appended with a file.
		{out: ".", want: "gitfs.go"},
		{out: "..", want: "../gitfs.go"},
		// A file in a dir stay the same.
		{out: "/tmp/f.go", want: "/tmp/f.go"},
	}
	for _, tt := range tests {
		t.Run(tt.out, func(t *testing.T) {
			got, err := getOut(tt.out)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetOut_fail(t *testing.T) {
	t.Parallel()
	tests := []string{
		// Output must be in existing directory.
		"nosuchdir/testout.go",
		// Output directory must exists.
		"nosuchdir/",
	}
	for _, out := range tests {
		t.Run(out, func(t *testing.T) {
			_, err := getOut(out)
			assert.Error(t, err)
		})
	}
}

func TestGetPkg(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pkg  string
		out  string
		want string
	}{
		{pkg: "foo", want: "foo"},
		{out: "", want: "main"},
		{out: ".", want: "main"},
		{out: "../../", want: "gitfs"},
	}
	for _, tt := range tests {
		t.Run(tt.out, func(t *testing.T) {
			got, err := getPkg(tt.pkg, tt.out)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func runGo(t *testing.T, args ...string) (stderr string, err error) {
	cmd := exec.Command("go", args...)
	stderrBuf, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, cmd.Start())
	stderrBytes, err := ioutil.ReadAll(stderrBuf)
	if err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	return string(stderrBytes), err
}

func clean() {
	paths, err := filepath.Glob("testout*.go")
	if err != nil {
		panic(err)
	}
	for _, path := range paths {
		os.Remove(path)
	}
}
