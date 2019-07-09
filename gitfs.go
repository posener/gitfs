// Package gitfs enables serving static file content from a git repository.
//
// It provides having static content in a Go project, not from Go files,
// but from any file in a project, submitted to a remote git repository.
// This content can also be versioned if required.
// For debug purposes, it enables loading this files from local path.
//
// Examples
//
// To setup a filesystem from a github repository `github.com/x/y` with
// at a given version v1.2.3, and path inside the project: "static":
//
// 	fs, err := gitfs.Open(ctx, "github.com/x/y/static@v1.2.3")
//
// Then, reading a file can be done by opening it:
//
// 	f, err := fs.Open("index.html")
//
// The variable fs, which is of type `http.FileSystem`, can be used to serve
// static content:
//
// 	http.Handle("/", http.FileServer(fs))
package gitfs

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/posener/gitfs/internal/githubfs"
	"github.com/posener/gitfs/internal/localfs"
	"github.com/posener/gitfs/log"
)

// OptClient sets up an HTTP client to perform request to the remote repository.
// This client can be used for authorization credentials.
func OptClient(client *http.Client) func(*config) {
	return func(c *config) {
		c.client = client
	}
}

// OptLocal result in looking for local git repository before accessing remote
// repository. The given path should be contained in a git repository which
// has a remote URL that matches the requested project.
func OptLocal(path string) func(*config) {
	return func(c *config) {
		c.localPath = path
	}
}

// New returns a new git filesystem for the given project.
//
// Github:
// If the given project is a github project (of the form github.com/<owner>/<repo>(@<ref>)?(#<path>)? ),
// the returned filesystem will be fetching files from the given project.
// ref is optional and can be any github ref:
//  * `heads/<branch name>` for a branch.
//  * `tags/<tag>` for releases or git tags.
//  * `<version>` for semver compatible releases (e.g. v1.2.3).
// If no ref is set, the default branch will be used.
func New(ctx context.Context, project string, opts ...option) (http.FileSystem, error) {
	var c config
	for _, opt := range opts {
		opt(&c)
	}

	switch {
	case c.localPath != "":
		return localfs.New(project, c.localPath)
	case githubfs.Match(project):
		return githubfs.New(ctx, c.client, project)
	default:
		return nil, errors.New("project type not supported")
	}
}

// SetLogger sets informative logging for gitfs. If nil, no logging
// will be done.
func SetLogger(logger log.Logger) {
	log.Log = logger
}

type config struct {
	client    *http.Client
	localPath string
}

type option func(*config)
