package fsutil

import (
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/pkg/errors"
	contentdiff "rsc.io/diff"
)

const (
	msgOnlyInA   = "only in a"
	msgOnlyInB   = "only in b"
	msgAFileBDir = "on a is file, on b directory"
	msgADirBFile = "on a is directory, on b file"
)

// FileSystemDiff lists all differences between two filesystems.
type FileSystemDiff []PathDiff

// PathDiff is a diff between two filesystems at a single path.
type PathDiff struct {
	Path string
	Diff string
}

// String returns pretty representation of a filesystem diff.
func (d FileSystemDiff) String() string {
	// Concatenate all differences.
	var out strings.Builder
	for _, line := range d {
		out.WriteString(line.Path + ": " + line.Diff + "\n")
	}
	return out.String()
}

// Diff returns the difference in filesystem structure and file content
// between two filesystems. If the implementation of the filesystem is
// different but the structure and content are equal, the function will
// consider the object as equal.
// For equal filesystems, an empty slice will be returned.
// The returned differences are ordered by file path.
func Diff(a, b http.FileSystem) (FileSystemDiff, error) {
	aFiles, err := lsR(a)
	if err != nil {
		return nil, errors.Errorf("walking filesystem a: %s", err)
	}
	bFiles, err := lsR(b)
	if err != nil {
		return nil, errors.Errorf("walking filesystem b: %s", err)
	}
	var d FileSystemDiff

	// Compare two slices of ordered file names. Always compare first element
	// in each slice and pop the elements from the slice accordingly.
	for len(aFiles) > 0 || len(bFiles) > 0 {
		switch {
		case len(bFiles) == 0 || (len(aFiles) > 0 && aFiles[0] < bFiles[0]):
			// File exists only in a.
			path := aFiles[0]
			d = append(d, PathDiff{Path: path, Diff: msgOnlyInA})
			aFiles = aFiles[1:]
		case len(aFiles) == 0 || (len(bFiles) > 0 && bFiles[0] < aFiles[0]):
			// File exists only in b.
			path := bFiles[0]
			d = append(d, PathDiff{Path: path, Diff: msgOnlyInB})
			bFiles = bFiles[1:]
		default:
			// File exists both in a and in b.
			path := aFiles[0]
			diff, err := contentDiff(a, b, path)
			if err != nil {
				return nil, err
			}
			if diff != "" {
				d = append(d, PathDiff{Path: path, Diff: diff})
			}
			aFiles = aFiles[1:]
			bFiles = bFiles[1:]
		}
	}
	return d, nil
}

// lsR is ls -r. Sorted by name.
func lsR(fs http.FileSystem) ([]string, error) {
	w := Walk(fs, "")
	var paths []string
	for w.Step() {
		paths = append(paths, w.Path())
	}
	if err := w.Err(); err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func contentDiff(a, b http.FileSystem, path string) (string, error) {
	aF, err := a.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "open %s in filesystem a", path)
	}
	defer aF.Close()

	bF, err := b.Open(path)
	if err != nil {
		return "", errors.Wrapf(err, "open %s in filesystem b", path)
	}
	defer bF.Close()

	aSt, err := aF.Stat()
	if err != nil {
		return "", errors.Wrapf(err, "stat %s in filesystem a", path)
	}

	bSt, err := bF.Stat()
	if err != nil {
		return "", errors.Wrapf(err, "stat %s in filesystem b", path)
	}

	if aSt.IsDir() || bSt.IsDir() {
		if !aSt.IsDir() {
			return msgAFileBDir, nil
		}
		if !bSt.IsDir() {
			return msgADirBFile, nil
		}
		return "", nil
	}

	aData, err := ioutil.ReadAll(aF)
	if err != nil {
		return "", errors.Wrapf(err, "reading %s from filesystem a", path)
	}

	bData, err := ioutil.ReadAll(bF)
	if err != nil {
		return "", errors.Wrapf(err, "reading %s from filesystem b", path)
	}

	d := contentdiff.Format(string(aData), string(bData))
	if d != "" {
		return "content diff:\n" + strings.TrimRight(d, "\n"), nil
	}
	return "", nil
}
