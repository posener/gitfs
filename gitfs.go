// Package gitfs enables using static files in Go code without packing them into binary.
//
// This package enable loading static files from a remote git repository,
// in Go code. This files can be used for static serving, template loading,
// or anything else.
// The following features are supported:
//
// * Loading of specific version is supported.
// * For debug purposes, the files can be loaded from local path. The transition
//   from remote project to local files is smooth.
// * Project is loaded instantly and files are loaded lazily but only once.
//
// Examples
//
// To setup a filesystem from a github repository `github.com/x/y` 
// at a given version v1.2.3 and path "static" inside the project:
//
// 	fs, err := gitfs.New(ctx, "github.com/x/y/static@v1.2.3")
//
// Then, reading a file can be done by opening it:
//
// 	f, err := fs.Open("index.html")
//
// The variable fs, which is of type `http.FileSystem`, can be used to serve
// static content:
//
// 	http.Handle("/", http.FileServer(fs))
//
// When used with private github repo, you would want to provide
// an HTTP client with the appropriate credentials. For example,
// if you have a Github Token in environnement variable `GITHUB_TOKEN`:
//  
// 	token := os.Getenv("GITHUB_TOKEN")
// 	client := oauth2.NewClient(
// 		context.Background(),
// 		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
// 	fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptClient(client))
//
// There are some useful functions in ./fsutil package. For example, it
// enables working easily with templates (They can work with any
// `http.FileSystem`):
//
// 	fs, err := gitfs.New(ctx, "github.com/x/y/templates")
// 	// Handle err...
// 	tmpl, err := fsutil.TmplParseGlob(fs, nil, "*.gotmpl")
// 	// Handle err...
// 	tmpl.ExecuteTemplate(...)
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
