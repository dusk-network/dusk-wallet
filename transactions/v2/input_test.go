package transactions

import (
	"bytes"
	"dusk-wallet/mlsag"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
)

func TestNewInput(t *testing.T) {
	var amount, mask, privkey, commToZero ristretto.Scalar
	amount.Rand()
	mask.Rand()
	privkey.Rand()
	commToZero.Rand()

	i := NewInput(amount, mask, privkey)

	for k := 1; k <= 10; k++ {

		i.AddDecoy(helper.RandomSlice(t, 32), generateDualKey(t))

		assert.Equal(t, len(i.baseInput.Offsets), k)
		assert.Equal(t, i.Proof.LenMembers(), k)
	}

	i.Proof.SetCommToZero(commToZero)

	sig, keyImage, err := i.Prove()
	assert.NotNil(t, sig)
	assert.NotNil(t, keyImage)
	assert.Nil(t, err)

	assert.True(t, bytes.Equal(keyImage.Bytes(), i.baseInput.KeyImage))

	buf := &bytes.Buffer{}
	err = sig.Encode(buf, false)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(buf.Bytes(), i.baseInput.Signature))
}

func generateDualKey(t *testing.T) mlsag.PubKeys {
	pubkeys := mlsag.PubKeys{}

	var primaryKey ristretto.Point
	primaryKey.Rand()
	pubkeys.AddPubKey(primaryKey)

	var secondaryKey ristretto.Point
	secondaryKey.Rand()
	pubkeys.AddPubKey(secondaryKey)

	return pubkeys
}
