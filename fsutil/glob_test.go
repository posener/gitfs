package fsutil

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pwd is the filesystem on which all tests run.
var pwd = http.Dir(".")

func TestGlobOpen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patterns   []string
		matches    []string
		notMatches []string
	}{
		{
			patterns:   []string{""},
			notMatches: []string{"testdata", "./testdata"},
		},
		{
			patterns: []string{"testdata"},
			matches:  []string{"testdata", "./testdata", "testdata/", "./testdata/"},
		},
		{
			patterns: []string{"", "testdata"},
			matches:  []string{"testdata"},
		},
		{
			patterns: []string{"testdata", ""},
			matches:  []string{"testdata"},
		},
		{
			patterns:   []string{"*/*1.gotmpl"},
			matches:    []string{"testdata/tmpl1.gotmpl", "./testdata/tmpl1.gotmpl", "./testdata/tmpl1.gotmpl/"},
			notMatches: []string{"testdata/tmpl2.gotmpl", "./testdata/tmpl2.gotmpl", "./testdata/tmpl2.gotmpl/"},
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.patterns, ":"), func(t *testing.T) {
			g, err := Glob(pwd, tt.patterns...)
			assert.NoError(t, err)
			for _, match := range tt.matches {
				t.Run("matches:"+match, func(t *testing.T) {
					_, err = g.Open(match)
					assert.NoError(t, err)
				})
			}
			for _, notMatch := range tt.notMatches {
				t.Run("not matches:"+notMatch, func(t *testing.T) {
					_, err = g.Open(notMatch)
					assert.EqualError(t, err, "file does not exist")
				})
			}
		})
	}
}

func TestGlobListDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patterns   []string
		open       string
		foundFiles []string
	}{
		{
			patterns: []string{"testdata"},
			open:     "testdata",
		},
		{
			patterns: []string{"", "testdata"},
			open:     "testdata",
		},
		{
			patterns: []string{"testdata", ""},
			open:     "testdata",
		},
		{
			patterns:   []string{"*/*1.gotmpl"},
			open:       "testdata",
			foundFiles: []string{"tmpl1.gotmpl"},
		},
		{
			patterns:   []string{"*/*.gotmpl"},
			open:       "testdata",
			foundFiles: []string{"tmpl1.gotmpl", "tmpl2.gotmpl"},
		},
		{
			// Extra part of path, there is no directory that fit this.
			patterns: []string{"*/*.gotmpl/*"},
			open:     "testdata",
		},
		{
			// No slash, only directory is available, but not the files in it.
			patterns: []string{"*"},
			open:     "testdata",
		},
		{
			// Matching a two components glob should match only directories.
			patterns:   []string{"*/*"},
			open:       ".",
			foundFiles: []string{"testdata"},
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.patterns, ":"), func(t *testing.T) {
			g, err := Glob(pwd, tt.patterns...)
			assert.NoError(t, err)
			dir, err := g.Open(tt.open)
			require.NoError(t, err)
			files, err := dir.Readdir(0)
			require.NoError(t, err)
			// Copy file names
			names := make([]string, 0, len(files))
			for _, file := range files {
				names = append(names, file.Name())
			}
			assert.ElementsMatch(t, names, tt.foundFiles)
		})
	}
}

func TestGlobOpenDir_failure(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patterns []string
		open     string
	}{
		{
			patterns: []string{""},
			open:     "testdata",
		},
		{
			patterns: []string{"*"},
			open:     "testdata1",
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.patterns, ":"), func(t *testing.T) {
			g, err := Glob(pwd, tt.patterns...)
			assert.NoError(t, err)
			_, err = g.Open(tt.open)
			require.Error(t, err)
		})
	}
}

func TestGlobReadDir_failure(t *testing.T) {
	t.Parallel()
	g, err := Glob(pwd, "*/*")
	assert.NoError(t, err)
	f, err := g.Open("testdata/tmpl1.gotmpl")
	require.NoError(t, err)
	// This is a file, so Readdir should fail
	_, err = f.Readdir(0)
	assert.Error(t, err)
}

func TestGlob_badPattern(t *testing.T) {
	t.Parallel()
	_, err := Glob(pwd, "[") // Missing closing bracket.
	assert.Error(t, err)
}

func TestGlob_noPattern(t *testing.T) {
	t.Parallel()
	g, err := Glob(pwd)
	require.NoError(t, err)
	assert.Equal(t, pwd, g)
}
