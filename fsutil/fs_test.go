package fsutil

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileSystem(t *testing.T) {
	t.Parallel()

	var got []string
	for walker := Walk(http.Dir("../internal"), "testdata"); walker.Step(); {
		got = append(got, walker.Path())
	}
	want := []string{
		"testdata",
		"testdata/f01",
		"testdata/d2",
		"testdata/d2/f21",
		"testdata/d1",
		"testdata/d1/d11",
		"testdata/d1/d11/f111",
	}
	assert.ElementsMatch(t, want, got)
}
