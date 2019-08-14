// Package gitfs is a complete solution for static files in Go code.
//
// When Go code uses non-Go files, they are not packaged into the binary.
// The common approach to the problem, as implemented by
// (go-bindata) https://github.com/jteeuwen/go-bindata
// is to convert all the required static files into Go code, which
// eventually compiled into the binary.
//
// This library takes a different approach, in which the static files are not
// required to be "binary-packed", and even no required to be in the same repository
// as the Go code. This package enables loading static content from a remote
// git repository, or packing it to the binary if desired or loaded
// from local path for development process. The transition from remote repository
// to binary packed content, to local content is completely smooth.
//
// *The API is simple and minimalistic*. The `New` method returns a (sub)tree
// of a Git repository, represented by the standard `http.FileSystem` interface.
// This object enables anything that is possible to do with a regular filesystem,
// such as opening a file or listing a directory.
// Additionally, the ./fsutil package provides enhancements over the `http.FileSystem`
// object (They can work with any object that implements the interface) such
// as loading Go templates in the standard way, walking over the filesystem,
// and applying glob patterns on a filesystem.
//
// Supported features:
//
// * Loading of specific version/tag/branch.
//
// * For debug purposes, the files can be loaded from local path instead of the
// remote repository.
//
// * Files are loaded lazily by default or they can be preloaded if required.
//
// * Files can be packed to the Go binary using a command line tool.
//
// * This project is using the standard `http.FileSystem` interface.
//
// * In ./fsutil there are some general useful tools around the
// `http.FileSystem` interace.
//
// Usage
//
// To create a filesystem using the `New` function, provide the Git
// project with the pattern: `github.com/<owner>/<repo>(/<path>)?(@<ref>)?`.
// If no `path` is specified, the root of the project will be used.
// `ref` can be any git branch using `heads/<branch name>` or any
// git tag using `tags/<tag>`. If the tag is of Semver format, the `tags/`
// prefix is not required. If no `ref` is specified, the default branch will
// be used.
//
// In the following example, the repository `github.com/x/y` at tag v1.2.3
// and internal path "static" is loaded:
//
// 	fs, err := gitfs.New(ctx, "github.com/x/y/static@v1.2.3")
//
// The variable `fs` implements the `http.FileSystem` interface.
// Reading a file from the repository can be done using the `Open` method.
// This function accepts a path, relative to the root of the defined
// filesystem.
//
// 	f, err := fs.Open("index.html")
//
// The `fs` variable can be used in anything that accept the standard interface.
// For example, it can be used for serving static content using the standard
// library:
//
// 	http.Handle("/", http.FileServer(fs))
//
// Private Repositories
//
// When used with private github repository, the Github API calls should be
// instrumented with the appropriate credentials. The credentials can be
// passed by providing an HTTP client.
//
// For example, to use a Github Token from environnement variable `GITHUB_TOKEN`:
//
// 	token := os.Getenv("GITHUB_TOKEN")
// 	client := oauth2.NewClient(
// 		context.Background(),
// 		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
// 	fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptClient(client))
//
// Development
//
// For quick development workflows, it is easier and faster to use local static
// content and not remote content that was pushed to a remote repository.
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
// them for you.
//
// To get the tool run: `go get github.com/posener/gitfs/cmd/gitfs`.
//
// Running the tool is by `gitfs <patterns>`. This generates a `gitfs.go`
// file in the current directory that contains all the used filesystems' data.
// This will cause all `gitfs.New` calls to automatically use the packed data,
// insted of fetching the data on runtime.
//
// By default, a test will also be generated with the code. This test fails
// when the local files are modified without updating the binary content.
//
// Use binary-packing with `go generate`: To generate all filesystems used
// by a project add `//go:generate gitfs ./...` in the root of the project.
// To generate only a specific filesystem add `//go:generate gitfs $GOFILE` in
// the file it is being used.
//
// An interesting anecdote is that gitfs command is using itself for generating
// its own templates.
//
// Excluding files
//
// Files exclusion can be done by including only specific files using a glob
// pattern with `OptGlob` option, using the Glob options. This will affect
// both local loading of files, remote loading and binary packing (may
// reduce binary size). For example:
//
// 	fs, err := gitfs.New(ctx,
// 		"github.com/x/y/templates",
// 		gitfs.OptGlob("*.gotmpl", "*/*.gotmpl"))
package gitfs

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/posener/gitfs/fsutil"
	"github.com/posener/gitfs/internal/binfs"
	"github.com/posener/gitfs/internal/githubfs"
	"github.com/posener/gitfs/internal/localfs"
	"github.com/posener/gitfs/internal/log"
)

// OptClient sets up an HTTP client to perform request to the remote repository.
// This client can be used for authorization credentials.
func OptClient(client *http.Client) option {
	return func(c *config) {
		c.client = client
	}
}

// OptLocal result in looking for local git repository before accessing remote
// repository. The given path should be contained in a git repository which
// has a remote URL that matches the requested project.
func OptLocal(path string) option {
	return func(c *config) {
		c.localPath = path
	}
}

// OptPrefetch sets prefetching all files in the filesystem when it is initially
// loaded.
func OptPrefetch(prefetch bool) option {
	return func(c *config) {
		c.prefetch = prefetch
	}
}

// OptGlob define glob patterns for which only matching files and directories
// will be included in the filesystem.
func OptGlob(patterns ...string) option {
	return func(c *config) {
		c.patterns = patterns
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
		fs, err := localfs.New(project, c.localPath)
		if err != nil {
			return nil, err
		}
		return fsutil.Glob(fs, c.patterns...)
	case binfs.Match(project):
		log.Printf("FileSystem %q from binary", project)
		return binfs.Get(project), nil
	case githubfs.Match(project):
		log.Printf("FileSystem %q from remote Github repository", project)
		return githubfs.New(ctx, c.client, project, c.prefetch, c.patterns)
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
	patterns  []string
}

type option func(*config)

type contexter interface {
	WithContext(ctx context.Context) http.File
}
