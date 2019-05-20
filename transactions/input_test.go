package transactions

import (
	"bytes"
	"dusk-wallet/mlsag"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestInputEncodeDecode(t *testing.T) {
	var amount ristretto.Scalar
	amount.Rand()

	input := generateInput(amount)
	proof := &mlsag.Proof{}
	proof.AddSecret(amount)

	pubKeys := mlsag.PubKeys{}
	var randPoint ristretto.Point
	randPoint.Rand()
	pubKeys.AddPubKey(randPoint)
	proof.AddDecoy(pubKeys)
	proof.AddDecoy(pubKeys)

	sig, keyImages, err := proof.Prove()
	assert.Nil(t, err)
	input.keyImages = keyImages
	input.Sig = sig

	buf := &bytes.Buffer{}
	err = input.Encode(buf)
	assert.Nil(t, err)

	var decodedInput Input
	err = decodedInput.Decode(buf)
	assert.Nil(t, err)

	ok := input.Equals(decodedInput)
	assert.True(t, ok)
}
