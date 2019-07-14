// Package gitfs loads static files in Go code without binary-packing.
//
// This package enable loading any file from remote git repository in Go code.
// This files can be used for static serving, template loading, content loading,
// or anything else.
//
// The following features are supported:
//
// * Loading of specific version is supported.
//
// * For debug purposes, the files can be loaded from local path. The transition
// from remote project to local files is smooth.
//
// * Project is loaded instantly and files are loaded lazily but only once.
//
// Usage
//
// First, we need to create a filesystem using the `New` function.
// This function accepts the project path with pattern:
// `github.com/<owner>/<repo>(/<path>)?(@<ref>)?`.
// If no `path` is specified, the root of the project will be used.
// `ref` can be any git branch using `heads/<branch name>` or any
// git tag using `tags/<tag>`. If the tag is of Semver format, the `tags/`
// prefix is not required.
// If no `ref` is specified, the default branch will be used.
//
// Here we will load repository `github.com/x/y` at version v1.2.3
// and internal path "static":
//
// 	fs, err := gitfs.New(ctx, "github.com/x/y/static@v1.2.3")
//
// Reading a file from the repository can be done using the `Open` method.
// This function accepts a path, relative to the root of the defined
// filesystem.
//
// 	f, err := fs.Open("index.html")
//
// The variable `fs` implements `http.FileSystem`, and can be used for anything
// that accepts it. For example, it can be used for serving static content
// using the standard library:
//
// 	http.Handle("/", http.FileServer(fs))
//
// When used with private github repository, it should be accessed with
// appropriate credentials. The credentials can be passed by providing an
// HTTP client. For example, to use a Github Token from environnement
// variable `GITHUB_TOKEN`:
//
// 	token := os.Getenv("GITHUB_TOKEN")
// 	client := oauth2.NewClient(
// 		context.Background(),
// 		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
// 	fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptClient(client))
//
// For debugging purposes, it is easier and faster to use local static
// content and not remote content that was pushed to the remote repository.
// This is enabled by the `OptLocal` option. To use this option only in
// local development and not in production system, it can be used as follow:
//
// 	local := os.Getenv("LOCAL_DEBUG")
// 	fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptLocal(local))
//
// Then, running the program with `LOCAL_DEBUG=.` will use local files while
// running without it will use the remote files. (the value of the environment
// variable should point to any directory within the github project).
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
//  * `<version>` for Semver compatible releases (e.g. v1.2.3).
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

// WithContext applies context to an http.File if it implements the
// contexter interface.
//
// Usage example:
//
// 	f, err := fs.Open("file")
// 	// Handle err...
// 	f = gitfs.WithContext(f, ctx)
// 	_, err = f.Read(...)
func WithContext(f http.File, ctx context.Context) http.File {
	fCtx, ok := f.(contexter)
	if !ok {
		return f
	}
	return fCtx.WithContext(ctx)
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

type contexter interface {
	WithContext(ctx context.Context) http.File
}
