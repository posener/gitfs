// An example that shows how gitfs helps using template files with Go code smoothly.
package main

import (
	"context"
	"log"
	"os"

	"github.com/posener/gitfs"
	"github.com/posener/gitfs/fsutil"
)

// Add debug mode environment variable. When running with `LOCAL_DEBUG=.`, the
// local git repository will be used instead of the remote github.
var localDebug = os.Getenv("LOCAL_DEBUG")

func main() {
	ctx := context.Background()
	// Open repository 'github.com/posener/gitfs' at path
	// 'examples/templates' with the local option from
	// environment variable.
	fs, err := gitfs.New(ctx,
		"github.com/posener/gitfs/examples/templates",
		gitfs.OptLocal(localDebug))
	if err != nil {
		log.Fatalf("Failed initializing git filesystem: %s.", err)
	}
	// Parse templates from the loaded filesystem using a glob
	// pattern.
	tmpls, err := fsutil.TmplParseGlob(fs, nil, "*.gotmpl")
	if err != nil {
		log.Fatalf("Failed parsing templates.")
	}
	// Execute a template according to its file name.
	tmpls.ExecuteTemplate(os.Stdout, "tmpl1.gotmpl", "Foo")
}
