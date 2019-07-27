package githubfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubNewProject(t *testing.T) {
	t.Parallel()
	tests := []struct {
		path string
		want project
	}{
		{
			path: "github.com/x/y@tags/v1",
			want: project{owner: "x", repo: "y", ref: "tags/v1"},
		},
		{
			path: "github.com/x/y@heads/foo",
			want: project{owner: "x", repo: "y", ref: "heads/foo"},
		},
		{
			path: "github.com/x/y",
			want: project{owner: "x", repo: "y", ref: ""},
		},
		{
			path: "github.com/x/y@v1",
			want: project{owner: "x", repo: "y", ref: "tags/v1"},
		},
		{
			path: "github.com/x/y@v1.2",
			want: project{owner: "x", repo: "y", ref: "tags/v1.2"},
		},
		{
			path: "github.com/x/y@v1.2.3",
			want: project{owner: "x", repo: "y", ref: "tags/v1.2.3"},
		},
		{
			path: "github.com/x/y@1",
			want: project{owner: "x", repo: "y", ref: "tags/1"},
		},
		{
			path: "github.com/x/y@1.2",
			want: project{owner: "x", repo: "y", ref: "tags/1.2"},
		},
		{
			path: "github.com/x/y@1.2.3",
			want: project{owner: "x", repo: "y", ref: "tags/1.2.3"},
		},
		{
			path: "github.com/x/y/static/path",
			want: project{owner: "x", repo: "y", path: "static/path/"},
		},
		{
			path: "github.com/x/y/static@v1.2.3",
			want: project{owner: "x", repo: "y", ref: "tags/v1.2.3", path: "static/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := newProject(tt.path)
			require.NoError(t, err)
			assert.Equal(t, &tt.want, got)
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
			p, err := newProject(path)
			assert.Error(t, err, "Got project=%+v", p)
		})
	}
}
