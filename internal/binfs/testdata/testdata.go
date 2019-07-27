// A dummy package for binfs testing purposes that creates two gitfs filesystems.
package main

import (
	"context"

	"github.com/posener/gitfs"
)

func main() {
	ctx := context.Background()
	gitfs.New(ctx, "github.com/a/b")
	gitfs.New(ctx, "github.com/c/d", gitfs.OptGlob("foo", "*"))
}
