package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pattern []string
		name    string
		isDir   bool
	}{
		// No pattern should match anything.
		{name: "foo"},
		{pattern: []string{"foo"}, name: "foo"},
		{pattern: []string{"*"}, name: "foo"},
		{pattern: []string{"foo"}, name: "./foo"},
		{pattern: []string{"foo"}, name: "foo/"},
		{pattern: []string{"foo"}, name: "./foo/"},
		{pattern: []string{"foo", "bar"}, name: "foo"},
		{pattern: []string{"bar", "foo"}, name: "foo"},
		{pattern: []string{"*/*"}, name: "foo/bar"},
		{pattern: []string{"*/*"}, name: "./foo/bar"},
		{pattern: []string{"*/*"}, name: "foo/bar/"},
		{pattern: []string{"*/*"}, name: "./foo/bar/"},
		{pattern: []string{"*/*"}, name: "foo", isDir: true},
		{pattern: []string{"*"}, name: "foo", isDir: true},
		{pattern: []string{"foo"}, name: "foo", isDir: true},
	}

	for _, tt := range tests {
		p, err := New(tt.pattern...)
		require.NoError(t, err)
		assert.True(t, p.Match(tt.name, tt.isDir))
	}
}

func TestMatch_noMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pattern []string
		name    string
		isDir   bool
	}{
		{pattern: []string{"f"}, name: "foo"},
		{pattern: []string{"f", "bar"}, name: "foo"},
		{pattern: []string{"bar", "f"}, name: "foo"},
		{pattern: []string{"*/*"}, name: "foo"},
		{pattern: []string{"*/*"}, name: "./foo"},
		{pattern: []string{"*/*"}, name: "foo/"},
		{pattern: []string{"*/*"}, name: "./foo/"},
		{pattern: []string{"*"}, name: "foo/bar"},
		{pattern: []string{"*"}, name: "./foo/bar"},
		{pattern: []string{"*"}, name: "foo/bar/"},
		{pattern: []string{"*"}, name: "./foo/bar/"},
		{pattern: []string{"*"}, name: "foo/bar", isDir: true},
		{pattern: []string{"*"}, name: "./foo/bar", isDir: true},
		{pattern: []string{"*"}, name: "foo/bar/", isDir: true},
		{pattern: []string{"*"}, name: "./foo/bar/", isDir: true},
	}

	for _, tt := range tests {
		p, err := New(tt.pattern...)
		require.NoError(t, err)
		assert.False(t, p.Match(tt.name, tt.isDir))
	}
}

func TestNew_badPattern(t *testing.T) {
	t.Parallel()
	_, err := New("[") // Missing closing bracket.
	assert.Error(t, err)
}
