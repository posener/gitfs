package localfs

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/posener/gitfs/internal/testfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	testfs.TestFS(t, func(t *testing.T, project string) (http.FileSystem, error) {
		return New(project, ".")
	})
}

func TestComputeSubdir(t *testing.T) {
	t.Parallel()
	gitRoot, err := lookupGitRoot(".")
	require.NoError(t, err)

	tests := []struct {
		project    string
		wantSubDir string
	}{
		// Simple case.
		{project: "github.com/posener/gitfs", wantSubDir: ""},
		// Any ref should be omitted.
		{project: "github.com/posener/gitfs@123", wantSubDir: ""},
		// With subdirectories.
		{project: "github.com/posener/gitfs/internal@123", wantSubDir: "internal"},
		{project: "github.com/posener/gitfs/internal/testdata", wantSubDir: "internal/testdata"},
	}
	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			subDir, err := computeSubdir(tt.project, gitRoot)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSubDir, subDir)
		})
	}
}

func TestComputeSubdir_failure(t *testing.T) {
	t.Parallel()
	gitRoot, err := lookupGitRoot(".")
	require.NoError(t, err)

	tests := []struct {
		project string
		path    string
	}{
		// Should not have a .git suffix.
		{project: "github.com/posener/gitfs.git", path: gitRoot},
		// Wrong domain.
		{project: "git.com/posener/gitfs", path: gitRoot},
		// Correct project but not a repository directory.
		{project: "github.com/posener/gitfs", path: "/tmp"},
	}

	for _, tt := range tests {
		t.Run(tt.project, func(t *testing.T) {
			_, err := computeSubdir(tt.project, tt.path)
			assert.Error(t, err)
		})
	}
}

func TestCleanRevision(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "x", cleanRevision("x"))
	assert.Equal(t, "x", cleanRevision("x@"))
	assert.Equal(t, "x", cleanRevision("x@v"))
}

func TestLookupGitRoot(t *testing.T) {
	t.Parallel()
	gitRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	// Check from current directory (not a git root)
	path, err := lookupGitRoot(".")
	require.NoError(t, err)
	assert.Equal(t, gitRoot, path)

	// Check from git root
	os.Chdir(gitRoot)
	path, err = lookupGitRoot(gitRoot)
	require.NoError(t, err)
	assert.Equal(t, gitRoot, path)

	// Check from /tmp - not a git repository
	path, err = lookupGitRoot("/tmp")
	assert.Error(t, err)
}
