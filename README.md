# gitfs

[![Build Status](https://travis-ci.org/posener/gitfs.svg?branch=master)](https://travis-ci.org/posener/gitfs)
[![codecov](https://codecov.io/gh/posener/gitfs/branch/master/graph/badge.svg)](https://codecov.io/gh/posener/gitfs)
[![golangci](https://golangci.com/badges/github.com/posener/gitfs.svg)](https://golangci.com/r/github.com/posener/gitfs)
[![GoDoc](https://godoc.org/github.com/posener/gitfs?status.svg)](http://godoc.org/github.com/posener/gitfs)
[![goreadme](https://goreadme.herokuapp.com/badge/posener/gitfs.svg)](https://goreadme.herokuapp.com)

Package gitfs is a complete solution for static files in Go code.

When Go code uses non-Go files, they are not packaged into the binary.
The common approach to the problem, as implemented by
[go-bindata](https://github.com/jteeuwen/go-bindata)
is to convert all the required static files into Go code, which
eventually compiled into the binary.

This library takes a different approach, in which the static files are not
required to be "binary-packed", and even no required to be in the same repository
as the Go code. This package enables loading static content from a remote
git repository, or packing it to the binary if desired or loaded
from local path for development process. The transition from remote repository
to binary packed content, to local content is completely smooth.

*The API is simple and minimalistic*. The `New` method returns a (sub)tree
of a Git repository, represented by the standard `http.FileSystem` interface.
This object enables anything that is possible to do with a regular filesystem,
such as opening a file or listing a directory.
Additionally, the [./fsutil](./fsutil) package provides enhancements over the `http.FileSystem`
object (They can work with any object that implements the interface) such
as loading Go templates in the standard way, walking over the filesystem,
and applying glob patterns on a filesystem.

Supported features:

* Loading of specific version/tag/branch.

* For debug purposes, the files can be loaded from local path instead of the
remote repository.

* Files are loaded lazily by default or they can be preloaded if required.

* Files can be packed to the Go binary using a command line tool.

* This project is using the standard `http.FileSystem` interface.

* In [./fsutil](./fsutil) there are some general useful tools around the
`http.FileSystem` interace.

#### Usage

To create a filesystem using the `New` function, provide the Git
project with the pattern: `github.com/<owner>/<repo>(/<path>)?(@<ref>)?`.
If no `path` is specified, the root of the project will be used.
`ref` can be any git branch using `heads/<branch name>` or any
git tag using `tags/<tag>`. If the tag is of Semver format, the `tags/`
prefix is not required. If no `ref` is specified, the default branch will
be used.

In the following example, the repository `github.com/x/y` at tag v1.2.3
and internal path "static" is loaded:

```go
fs, err := gitfs.New(ctx, "github.com/x/y/static@v1.2.3")
```

The variable `fs` implements the `http.FileSystem` interface.
Reading a file from the repository can be done using the `Open` method.
This function accepts a path, relative to the root of the defined
filesystem.

```go
f, err := fs.Open("index.html")
```

The `fs` variable can be used in anything that accept the standard interface.
For example, it can be used for serving static content using the standard
library:

```go
http.Handle("/", http.FileServer(fs))
```

#### Private Repositories

When used with private github repository, the Github API calls should be
instrumented with the appropriate credentials. The credentials can be
passed by providing an HTTP client.

For example, to use a Github Token from environnement variable `GITHUB_TOKEN`:

```go
token := os.Getenv("GITHUB_TOKEN")
client := oauth2.NewClient(
	context.Background(),
	oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptClient(client))
```

#### Development

For quick development workflows, it is easier and faster to use local static
content and not remote content that was pushed to a remote repository.
This is enabled by the `OptLocal` option. To use this option only in
local development and not in production system, it can be used as follow:

```go
local := os.Getenv("LOCAL_DEBUG")
fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptLocal(local))
```

In this example, we stored the value for `OptLocal` in an environment
variable. As a result, when running the program with `LOCAL_DEBUG=.`
local files will be used, while running without it will result in using
the remote files. (the value of the environment variable should point
to any directory within the github project).

#### Binary Packing

Using gitfs does not mean that files are required to be remotely fetched.
When binary packing of the files is needed, a command line tool can pack
them for you.

To get the tool run: `go get github.com/posener/gitfs/cmd/gitfs`.

Running the tool is by `gitfs <patterns>`. This generates a `gitfs.go`
file in the current directory that contains all the used filesystems' data.
This will cause all `gitfs.New` calls to automatically use the packed data,
insted of fetching the data on runtime.

## Sub Packages

* [bin](./bin): Package bin is a proxy to the internal/binfs.Register function.

* [fsutil](./fsutil): Package fsutil provides useful utility functions for http.FileSystem.

#### Examples

##### Fsutil

The [./fsutil](./fsutil) package is a collection of useful functions that can work with
any `http.FileSystem` implementation.
For example, here we will use a function that loads go templates from the
filesystem.

```golang
ctx := context.Background()

// Open a git remote repository `posener/gitfs` in path `examples/templates`.
fs, err := New(ctx, "github.com/posener/gitfs/examples/templates")
if err != nil {
    log.Fatalf("Failed initialize filesystem: %s", err)
}

// Use util function that loads all templates according to a glob pattern.
tmpls, err := fsutil.TmplParseGlob(fs, nil, "*.gotmpl")
if err != nil {
    log.Fatalf("Failed parsing templates: %s", err)
}

// Execute the template and write to stdout.
tmpls.ExecuteTemplate(os.Stdout, "tmpl1.gotmpl", "Foo")
```

##### Open

With gitfs you can open a remote git repository, and load any file,
including non-go files.
In this example, the README.md file of a remote repository is loaded.

```golang
ctx := context.Background()

// The load path is of the form: github.com/<owner>/<repo>(/<path>)?(@<ref>)?.
// `ref` can reference any git tag or branch. If github releases are in Semver format,
// the `tags/` prefix is not needed in the `ref` part.
fs, err := New(ctx, "github.com/kelseyhightower/helloworld@3.0.0")
if err != nil {
    log.Fatalf("Failed initialize filesystem: %s", err)
}

// Open any file in the github repository, using the `Open` function. Both files
// and directory can be opened. The content is not loaded until it is actually being
// read. The content is loaded only once.
f, err := fs.Open("README.md")
if err != nil {
    log.Fatalf("Failed opening file: %s", err)
}

// Copy the content to stdout.
io.Copy(os.Stdout, f)
```


---

Created by [goreadme](https://github.com/apps/goreadme)
