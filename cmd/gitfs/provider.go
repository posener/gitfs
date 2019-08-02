package main

import (
	"context"
	"net/http"

	"github.com/posener/gitfs"
	"github.com/posener/gitfs/internal/binfs"
)

func provider(c binfs.Config) (http.FileSystem, error) {
	return gitfs.New(context.Background(), c.Project,
		gitfs.OptPrefetch(true), gitfs.OptLocal("."), gitfs.OptGlob(c.GlobPatterns()...))
}
