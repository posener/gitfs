package binfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegister_illegalVersion(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() { Register("github.com/x/y", EncodeVersion+1, "") })
}
