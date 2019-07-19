// Package gitfs is a complete solution for static files in Go code.
//
// This package enables loading any file from remote git repository in Go code.
// The static files are not required to be packed into the go binary in order to
// be used. They can be loaded in runtime when accessed, or preloaded when the
// program starts.
//
// The following features are supported:
//
// * Loading of specific version/tag/branch.
//
// * For debug purposes, the files can be loaded from local path instead of the
// remote repository. The transition from remote project to local files is smooth.
//
// * Files are loaded lazily by default or they can be preloaded if required.
//
// * Files can be packed to the Go binary using a command line tool, which provides
// a smooth transition from using remote repository to binary packed.
//
// * This project is using the standard `http.FileSystem` interface for the loaded
// static files, and provides tooling around it.
//
// Usage
//
// First, create a filesystem using the `New` function. This function accepts the
// project path with pattern: `github.com/<owner>/<repo>(/<path>)?(@<ref>)?`.
// If no `path` is specified, the root of the project will be used.
// `ref` can be any git branch using `heads/<branch name>` or any
// git tag using `tags/<tag>`. If the tag is of Semver format, the `tags/`
// prefix is not required.
// If no `ref` is specified, the default branch will be used.
//
// In the following example, the repository `github.com/x/y` at version v1.2.3
// and internal path "static" is loaded:
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
// Private Repositories
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
// Development
//
// For quick development purposes, it is easier and faster to use local static
// content and not remote content that was pushed to the remote repository.
// This is enabled by the `OptLocal` option. To use this option only in
// local development and not in production system, it can be used as follow:
//
// 	local := os.Getenv("LOCAL_DEBUG")
// 	fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptLocal(local))
//
// In this example, we stored the value for `OptLocal` in an environment
// variable. As a result, when running the program with `LOCAL_DEBUG=.`
// local files will be used, while running without it will result in using
// the remote files. (the value of the environment variable should point
// to any directory within the github project).
//
// Binary Packing
//
// Using gitfs does not mean that files are required to be remotely fetched.
// When binary packing of the files is needed, a command line tool can pack
// them and no other changes in the code are required.
//
// To get the tool run: `go get github.com/posener/gitfs/cmd/gitfs`.
//
// Running the tool is by `gitfs <patterns>`. It will generate A file in the
// current directory that will contain all the required filesystem.
// The filesystems are detected by searching for `gitfs.New` calls.
package gitfs

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/posener/gitfs/internal/binfs"
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

// OptPrefetch sets prefetching all files in the filesystem when it is initially
// loaded.
func OptPrefetch(prefetch bool) func(*config) {
	return func(c *config) {
		c.prefetch = prefetch
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
		log.Printf("FileSystem %q from local directory", project)
		return localfs.New(project, c.localPath)
	case binfs.Match(project):
		log.Printf("FileSystem %q from binary", project)
		return binfs.Get(project), nil
	case githubfs.Match(project):
		log.Printf("FileSystem %q from remote Github repository", project)
		return githubfs.New(ctx, c.client, project, c.prefetch)
	default:
		return nil, errors.Errorf("project %q not supported", project)
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
	prefetch  bool
}

type option func(*config)

type contexter interface {
	WithContext(ctx context.Context) http.File
}
