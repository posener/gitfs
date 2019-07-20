// Package bin is a proxy to the internal/binfs.Register function.
//
// It is used by the gitfs command line.
package bin

import "github.com/posener/gitfs/internal/binfs"

// Register registers binary data of a given project.
func Register(project string, version int, data string) {
	binfs.Register(project, version, data)
}
