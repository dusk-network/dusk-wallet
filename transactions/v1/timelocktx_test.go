package transactions

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimeLockEncodeDecode(t *testing.T) {
	tl, err := NewTimeLock(1, 10, 1000)
	assert.Nil(t, err)

	buf := &bytes.Buffer{}
	err = tl.Encode(buf)
	assert.Nil(t, err)

}
