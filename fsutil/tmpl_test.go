package fsutil

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTmplParse(t *testing.T) {
	t.Parallel()
	fs := http.Dir(".")
	buf := bytes.NewBuffer(nil)

	tmpl, err := TmplParse(fs, nil, "testdata/tmpl1.gotmpl", "testdata/tmpl2.gotmpl")
	require.NoError(t, err)

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl1.gotmpl", "foo"))
	assert.Equal(t, "hello, foo", buf.String())

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl2.gotmpl", "foo"))
	assert.Equal(t, "hi, foo", buf.String())
}

func TestTmplParse_noSuchFile(t *testing.T) {
	t.Parallel()
	fs := http.Dir(".")
	_, err := TmplParse(fs, nil, "testdata/tmpl1.gotmpl", "testdata/nosuchfile")
	assert.Error(t, err)
	_, err = TmplParse(fs, nil, "testdata/nosuchfile", "testdata/tmpl1.gotmpl")
	assert.Error(t, err)
}

func TestTmplParse_emptyFileNames(t *testing.T) {
	t.Parallel()
	fs := http.Dir(".")
	_, err := TmplParse(fs, nil)
	assert.Error(t, err)
}

func TestTmplParseGlob(t *testing.T) {
	t.Parallel()
	buf := bytes.NewBuffer(nil)
	fs := http.Dir(".")

	// Match all files in the directory.
	tmpl, err := TmplParseGlob(fs, nil, "testdata/*.gotmpl")
	require.NoError(t, err)

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl1.gotmpl", "foo"))
	assert.Equal(t, "hello, foo", buf.String())

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl2.gotmpl", "foo"))
	assert.Equal(t, "hi, foo", buf.String())

	// Match only one file.
	tmpl, err = TmplParseGlob(fs, nil, "testdata/tmpl1.*")
	require.NoError(t, err)

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl1.gotmpl", "foo"))
	assert.Equal(t, "hello, foo", buf.String())

	buf.Reset()
	assert.Error(t, tmpl.ExecuteTemplate(buf, "tmpl2.gotmpl", "foo"))
}

func TestTmplParseHTML(t *testing.T) {
	t.Parallel()
	fs := http.Dir(".")
	tmpl, err := TmplParseHTML(fs, nil, "testdata/tmpl1.gotmpl", "testdata/tmpl2.gotmpl")
	require.NoError(t, err)
	buf := bytes.NewBuffer(nil)
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl1.gotmpl", "foo"))
	assert.Equal(t, "hello, foo", buf.String())

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl2.gotmpl", "foo"))
	assert.Equal(t, "hi, foo", buf.String())
}

func TestTmplParseGlobHTML(t *testing.T) {
	t.Parallel()
	buf := bytes.NewBuffer(nil)
	fs := http.Dir(".")

	// Match all files in the directory.
	tmpl, err := TmplParseGlobHTML(fs, nil, "testdata/*.gotmpl")
	require.NoError(t, err)

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl1.gotmpl", "foo"))
	assert.Equal(t, "hello, foo", buf.String())

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl2.gotmpl", "foo"))
	assert.Equal(t, "hi, foo", buf.String())

	// Match only one file.
	tmpl, err = TmplParseGlobHTML(fs, nil, "testdata/tmpl1.*")
	require.NoError(t, err)

	buf.Reset()
	require.NoError(t, tmpl.ExecuteTemplate(buf, "tmpl1.gotmpl", "foo"))
	assert.Equal(t, "hello, foo", buf.String())

	buf.Reset()
	assert.Error(t, tmpl.ExecuteTemplate(buf, "tmpl2.gotmpl", "foo"))
}

func TestTmplParseHTML_noSuchFile(t *testing.T) {
	t.Parallel()
	fs := http.Dir(".")
	_, err := TmplParseHTML(fs, nil, "testdata/tmpl1.gotmpl", "testdata/nosuchfile")
	assert.Error(t, err)
	_, err = TmplParseHTML(fs, nil, "testdata/nosuchfile", "testdata/tmpl1.gotmpl")
	assert.Error(t, err)
}

func TestTmplParseHTML_emptyFileNames(t *testing.T) {
	t.Parallel()
	fs := http.Dir(".")
	_, err := TmplParseHTML(fs, nil)
	assert.Error(t, err)
}
