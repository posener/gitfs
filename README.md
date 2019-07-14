# gitfs

[![Build Status](https://travis-ci.org/posener/gitfs.svg?branch=master)](https://travis-ci.org/posener/gitfs)
[![codecov](https://codecov.io/gh/posener/gitfs/branch/master/graph/badge.svg)](https://codecov.io/gh/posener/gitfs)
[![golangci](https://golangci.com/badges/github.com/posener/gitfs.svg)](https://golangci.com/r/github.com/posener/gitfs)
[![GoDoc](https://godoc.org/github.com/posener/gitfs?status.svg)](http://godoc.org/github.com/posener/gitfs)
[![goreadme](https://goreadme.herokuapp.com/badge/posener/gitfs.svg)](https://goreadme.herokuapp.com)

Package gitfs loads static files in Go code without binary-packing.

This package enable loading any file from remote git repository in Go code.
This files can be used for static serving, template loading, content loading,
or anything else.

The following features are supported:

* Loading of specific version is supported.

* For debug purposes, the files can be loaded from local path. The transition
from remote project to local files is smooth.

* Project is loaded instantly and files are loaded lazily but only once.

#### Usage

First, we need to create a filesystem using the `New` function.
This function accepts the project path with pattern:
`github.com/<owner>/<repo>(/<path>)?(@<ref>)?`.
If no `path` is specified, the root of the project will be used.
`ref` can be any git branch using `heads/<branch name>` or any
git tag using `tags/<tag>`. If the tag is of Semver format, the `tags/`
prefix is not required.
If no `ref` is specified, the default branch will be used.

Here we will load repository `github.com/x/y` at version v1.2.3
and internal path "static":

```go
fs, err := gitfs.New(ctx, "github.com/x/y/static@v1.2.3")
```

Reading a file from the repository can be done using the `Open` method.
This function accepts a path, relative to the root of the defined
filesystem.

```go
f, err := fs.Open("index.html")
```

The variable `fs` implements `http.FileSystem`, and can be used for anything
that accepts it. For example, it can be used for serving static content
using the standard library:

```go
http.Handle("/", http.FileServer(fs))
```

When used with private github repository, it should be accessed with
appropriate credentials. The credentials can be passed by providing an
HTTP client. For example, to use a Github Token from environnement
variable `GITHUB_TOKEN`:

```go
token := os.Getenv("GITHUB_TOKEN")
client := oauth2.NewClient(
	context.Background(),
	oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token}))
fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptClient(client))
```

For debugging purposes, it is easier and faster to use local static
content and not remote content that was pushed to the remote repository.
This is enabled by the `OptLocal` option. To use this option only in
local development and not in production system, it can be used as follow:

```go
local := os.Getenv("LOCAL_DEBUG")
fs, err := gitfs.New(ctx, "github.com/x/y", gitfs.OptLocal(local))
```

Then, running the program with `LOCAL_DEBUG=.` will use local files while
running without it will use the remote files. (the value of the environment
variable should point to any directory within the github project).

## Sub Packages

* [fsutil](./fsutil): Package fsutil provides useful utility functions for http.FileSystem.

* [log](./log)

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
