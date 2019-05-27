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
	input.Proof.SetPrimaryKey(amount)
	input.Proof.SetCommToZero(amount)

	pubKeys := mlsag.PubKeys{}
	var randPoint ristretto.Point
	randPoint.Rand()
	pubKeys.AddPubKey(randPoint)
	pubKeys.AddPubKey(randPoint)

	input.Proof.AddDecoy(pubKeys)
	input.Proof.AddDecoy(pubKeys)

	err := input.Prove()
	assert.Nil(t, err)

	buf := &bytes.Buffer{}
	err = input.Encode(buf)
	assert.Nil(t, err)

	var decodedInput Input
	err = decodedInput.Decode(buf)
	assert.Nil(t, err)

	ok := input.Equals(decodedInput)
	assert.True(t, ok)
}
