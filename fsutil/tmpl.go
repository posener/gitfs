package fsutil

import (
	"bytes"
	htmltmpl "html/template"
	"net/http"
	"path/filepath"
	"strings"
	txttmpl "text/template"

	"github.com/pkg/errors"
)

// TmplParse parses templates from the given filesystem according to the
// given paths. If tmpl is not nil, the templates will be added to it.
// paths must contain at least one path. All paths must exist in the
// given filesystem.
func TmplParse(fs http.FileSystem, tmpl *txttmpl.Template, paths ...string) (*txttmpl.Template, error) {
	t := tmplParser{Template: tmpl}
	err := parseFiles(fs, t.parse, paths...)
	return t.Template, err
}

// TmplParseGlob parses templates from the given filesystem according to
// the provided glob pattern. If tmpl is not nil, the templates will be
// added to it.
func TmplParseGlob(fs http.FileSystem, tmpl *txttmpl.Template, pattern string) (*txttmpl.Template, error) {
	t := tmplParser{Template: tmpl}
	err := parseGlob(fs, t.parse, pattern)
	return t.Template, err
}

// TmplParseHTML parses HTML templates from the given filesystem according
// to the given paths. If tmpl is not nil, the templates will be added to
// it. paths must contain at least one path. All paths must exist in the
// given filesystem.
func TmplParseHTML(fs http.FileSystem, tmpl *htmltmpl.Template, paths ...string) (*htmltmpl.Template, error) {
	t := tmplParserHTML{Template: tmpl}
	err := parseFiles(fs, t.parse, paths...)
	return t.Template, err
}

// TmplParseGlobHTML parses HTML templates from the given filesystem
// according to the provided glob pattern. If tmpl is not nil, the
// templates will be added to it.
func TmplParseGlobHTML(fs http.FileSystem, tmpl *htmltmpl.Template, pattern string) (*htmltmpl.Template, error) {
	t := tmplParserHTML{Template: tmpl}
	err := parseGlob(fs, t.parse, pattern)
	return t.Template, err
}

type tmplParser struct {
	*txttmpl.Template
}

func (t *tmplParser) parse(name, content string) error {
	var err error
	if t.Template == nil {
		t.Template = txttmpl.New(name)
	} else {
		t.Template = t.New(name)
	}
	t.Template, err = t.Parse(content)
	return err
}

type tmplParserHTML struct {
	*htmltmpl.Template
}

func (t *tmplParserHTML) parse(name, content string) error {
	var err error
	if t.Template == nil {
		t.Template = htmltmpl.New(name)
	} else {
		t.Template = t.New(name)
	}
	t.Template, err = t.Parse(content)
	return err
}

func parseFiles(fs http.FileSystem, parse func(name string, content string) error, paths ...string) error {
	if len(paths) == 0 {
		return errors.New("no paths provided")
	}
	buf := bytes.NewBuffer(nil)
	for _, path := range paths {
		f, err := fs.Open(strings.Trim(path, "/"))
		if err != nil {
			return errors.Wrapf(err, "opening template %s", path)
		}
		name := filepath.Base(path)
		buf.Reset()
		buf.ReadFrom(f)
		err = parse(name, buf.String())
		if err != nil {
			return errors.Wrapf(err, "parsing template %s", path)
		}
	}
	return nil
}

func parseGlob(fs http.FileSystem, parse func(name string, content string) error, pattern string) error {
	buf := bytes.NewBuffer(nil)
	walker := Walk(fs, "")
	for walker.Step() {
		matched, err := filepath.Match(pattern, walker.Path())
		if err != nil {
			return err
		}
		if !matched {
			continue
		}

		f, err := fs.Open(walker.Path())
		if err != nil {
			return errors.Wrapf(err, "opening template %s", walker.Path())
		}
		st, err := f.Stat()
		if err != nil {
			return errors.Wrapf(err, "stat %s", walker.Path())
		}
		if st.IsDir() {
			continue
		}

		buf.Reset()
		buf.ReadFrom(f)
		err = parse(walker.Stat().Name(), buf.String())
		if err != nil {
			return errors.Wrapf(err, "parsing template %s", walker.Path())
		}
	}
	if err := walker.Err(); err != nil {
		return errors.Wrap(err, "failed walking filesystem")
	}
	return nil
}
