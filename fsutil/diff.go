package fsutil

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"rsc.io/diff"
)

const (
	msgOnlyInA     = "only in {{.A}}"
	msgOnlyInB     = "only in {{.B}}"
	msgAFileBDir   = "on {{.A}} is file, on {{.B}} directory"
	msgADirBFile   = "on {{.A}} is directory, on {{.B}} file"
	msgContentDiff = "content diff (-{{.A}}, +{{.B}}):"
)

// FileSystemDiff lists all differences between two filesystems.
type FileSystemDiff struct {
	Diffs []PathDiff
	// FileSystem names.
	A, B string
}

// PathDiff is a diff between two filesystems at a single path.
type PathDiff struct {
	Path     string
	Diff     string
	DiffInfo string
}

func (d *FileSystemDiff) template(tmpl string) string {
	out := bytes.NewBuffer(nil)
	err := template.Must(template.New("title").Parse(tmpl)).Execute(out, d)
	if err != nil {
		panic(err)
	}
	return out.String()
}

// String returns pretty representation of a filesystem diff.
func (d *FileSystemDiff) String() string {
	if len(d.Diffs) == 0 {
		return ""
	}
	// Concatenate all differences.
	out := strings.Builder{}
	out.WriteString(d.template("Diff between {{.A}} and {{.B}}:\n"))
	for _, diff := range d.Diffs {
		out.WriteString("[" + diff.Path + "]: " + d.template(diff.Diff) + "\n")
		if diff.DiffInfo != "" {
			out.WriteString(diff.DiffInfo + "\n")
		}
	}
	return out.String()
}

// Diff returns the difference in filesystem structure and file content
// between two filesystems. If the implementation of the filesystem is
// different but the structure and content are equal, the function will
// consider the object as equal.
// For equal filesystems, an empty slice will be returned.
// The returned differences are ordered by file path.
func Diff(a, b http.FileSystem) (*FileSystemDiff, error) {
	aFiles, err := lsR(a)
	if err != nil {
		return nil, errors.Errorf("walking filesystem a: %s", err)
	}
	bFiles, err := lsR(b)
	if err != nil {
		return nil, errors.Errorf("walking filesystem b: %s", err)
	}

	d := &FileSystemDiff{A: "a", B: "b"}
	// Compare two slices of ordered file names. Always compare first element
	// in each slice and pop the elements from the slice accordingly.
	for len(aFiles) > 0 || len(bFiles) > 0 {
		switch {
		case len(bFiles) == 0 || (len(aFiles) > 0 && aFiles[0] < bFiles[0]):
			// File exists only in a.
			path := aFiles[0]
			d.Diffs = append(d.Diffs, PathDiff{Path: path, Diff: msgOnlyInA})
			aFiles = aFiles[1:]
		case len(aFiles) == 0 || (len(bFiles) > 0 && bFiles[0] < aFiles[0]):
			// File exists only in b.
			path := bFiles[0]
			d.Diffs = append(d.Diffs, PathDiff{Path: path, Diff: msgOnlyInB})
			bFiles = bFiles[1:]
		default:
			// File exists both in a and in b.
			path := aFiles[0]
			diff, err := contentDiff(a, b, path)
			if err != nil {
				return nil, err
			}
			if diff != nil {
				d.Diffs = append(d.Diffs, *diff)
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

func contentDiff(a, b http.FileSystem, path string) (*PathDiff, error) {
	aF, err := a.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "open %s in filesystem a", path)
	}
	defer aF.Close()

	bF, err := b.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "open %s in filesystem b", path)
	}
	defer bF.Close()

	aSt, err := aF.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "stat %s in filesystem a", path)
	}

	bSt, err := bF.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "stat %s in filesystem b", path)
	}

	if aSt.IsDir() || bSt.IsDir() {
		if !aSt.IsDir() {
			return &PathDiff{Path: path, Diff: msgAFileBDir}, nil
		}
		if !bSt.IsDir() {
			return &PathDiff{Path: path, Diff: msgADirBFile}, nil
		}
		return nil, nil
	}

	aData, err := ioutil.ReadAll(aF)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s from filesystem a", path)
	}

	bData, err := ioutil.ReadAll(bF)
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s from filesystem b", path)
	}

	if string(aData) == string(bData) {
		return nil, nil
	}
	d := diff.Format(string(aData), string(bData), diff.OptSuppressCommon())
	if d != "" {
		return &PathDiff{
			Path:     path,
			Diff:     msgContentDiff,
			DiffInfo: strings.TrimRight(d, "\n"),
		}, nil
	}
	return nil, nil
}
