package transactions

import (
	"bytes"
	"dusk-wallet/key"
	"math/rand"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestOutputEncodeDecode(t *testing.T) {
	var r, amount ristretto.Scalar
	r.Rand()
	amount.Rand()
	r32 := rand.Uint32()
	keyPair := key.NewKeyPair([]byte("seed for test"))

	output, err := NewOutput(r, amount, r32, *keyPair.PublicKey())
	assert.Nil(t, err)

	buf := &bytes.Buffer{}
	err = output.Encode(buf)
	assert.Nil(t, err)

	var decodedOutput Output
	err = decodedOutput.Decode(buf)
	assert.Nil(t, err)

	ok := output.Equals(decodedOutput)
	assert.True(t, ok)
}
