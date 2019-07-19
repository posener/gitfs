// Package binfs is filesystem over registered binary data.
//
// This pacakge is used by ./cmd/gitfs to generate files that
// contain static content of a filesystem.
package binfs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"github.com/posener/gitfs/fsutil"
	"github.com/posener/gitfs/internal/tree"
)

// EncodeVersion is the current encoding version.
const EncodeVersion = 1

// data maps registered projects (through `Register()` call)
// to the corresponding filesystem that they represent.
var data map[string]http.FileSystem

// fsStorage stores all filesystem structure and all file contents.
type fsStorage struct {
	// Files maps all file paths from root of the filesystem to
	// their contents.
	Files map[string][]byte
	// Dirs is the set of paths of directories in the filesystem.
	Dirs map[string]bool
}

func init() {
	data = make(map[string]http.FileSystem)
	gob.Register(fsStorage{})
}

// Register a filesystem under the project name.
// It panics if anything goes wrong.
func Register(project string, version int, encoded string) {
	if data[project] != nil {
		panic(fmt.Sprintf("Project %s registered multiple times", project))
	}
	var (
		fs  http.FileSystem
		err error
	)
	switch version {
	case 1:
		fs, err = decodeV1(encoded)
	default:
		panic(fmt.Sprintf(`Registered filesystem is from future version %d.
			The current gitfs suports version up to %d.
			Please update github.com/posener/gitfs.`, version, EncodeVersion))
	}
	if err != nil {
		panic(fmt.Sprintf("Failed decoding project %q: %s", project, err))
	}
	data[project] = fs
}

// Match returns wether project exists in registered binaries.
// The matching is done also over the project `ref`.
func Match(project string) bool {
	_, ok := data[project]
	return ok
}

// Get returns filesystem of a registered project.
func Get(project string) http.FileSystem {
	return data[project]
}

// encode converts a filesystem to an encoded string. All filesystem structure
// and file content is stored.
//
// Note: modifying this function should probably increase EncodeVersion const,
// and should probably add a new `decode` function for the new version.
func encode(fs http.FileSystem) (string, error) {
	// storage is an object that contains all filesystem information.
	storage := newFSStorage()

	// Walk the provided filesystem, and add all its content to storage.
	walker := fsutil.Walk(fs, "")
	for walker.Step() {
		path := walker.Path()
		if path == "" {
			continue
		}
		if walker.Stat().IsDir() {
			storage.Dirs[path] = true
		} else {
			b, err := readFile(fs, path)
			if err != nil {
				return "", err
			}
			storage.Files[path] = b
		}
	}
	if err := walker.Err(); err != nil {
		return "", errors.Wrap(err, "walking filesystem")
	}

	// Encode the storage object into a string.
	b := bytes.NewBuffer(nil)
	err := gob.NewEncoder(b).Encode(storage)
	if err != nil {
		return "", errors.Wrap(err, "encoding gob")
	}
	return base64.StdEncoding.EncodeToString(b.Bytes()), nil
}

// decodeV1 returns a filesystem from data that was encoded in V1.
func decodeV1(data string) (tree.Tree, error) {
	var storage fsStorage
	b, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, errors.Wrap(err, "decoding base64")
	}
	err = gob.NewDecoder(bytes.NewReader(b)).Decode(&storage)
	if err != nil {
		return nil, errors.Wrap(err, "decoding gob")
	}
	t := make(tree.Tree)
	for dir := range storage.Dirs {
		t.AddDir(dir)
	}
	for path, content := range storage.Files {
		content := content
		t.AddFile(path, len(content), func(ctx context.Context) ([]byte, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return content, nil
		})
	}
	return t, err
}

// readFile is a utility function that reads content of the file
// denoted by path from the provided filesystem.
func readFile(fs http.FileSystem, path string) ([]byte, error) {
	f, err := fs.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "opening file %s", path)
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrapf(err, "reading file content %s", path)
	}
	return b, nil
}

func newFSStorage() fsStorage {
	return fsStorage{
		Files: make(map[string][]byte),
		Dirs:  make(map[string]bool),
	}
}
