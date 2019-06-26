package transactions

import (
	"dusk-wallet/key"
	"math/rand"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestNewOutput(t *testing.T) {
	var r, amount ristretto.Scalar
	r.Rand()
	amount.Rand()
	r32 := rand.Uint32()
	keyPair := key.NewKeyPair([]byte("seed for test"))

	out := NewOutput(r, amount, r32, *keyPair.PublicKey())

	var R ristretto.Point
	R.ScalarMultBase(&r)

	_, ok := keyPair.DidReceiveTx(R, out.PubKey, r32)
	assert.True(t, ok)

	assert.Equal(t, out.amount, amount)
	assert.Equal(t, out.Index, r32)
}

func TestEncryptionAmount(t *testing.T) {
	keyPair := key.NewKeyPair([]byte("this is the seed"))
	var amount, r ristretto.Scalar
	amount.Rand()
	r.Rand()

	var R ristretto.Point
	R.ScalarMultBase(&r)

	pvKey, err := keyPair.PrivateView()
	assert.Nil(t, err)
	
	encryptedAmount := encryptAmount(amount, r, 0, *keyPair.PublicKey().PubView)
	decryptedAmount := decryptAmount(encryptedAmount, R, 0, *pvKey)
	
	assert.Equal(t, decryptedAmount, amount)
}

func TestEncryptionMask(t *testing.T) {
	keyPair := key.NewKeyPair([]byte("this is the seed"))
	var mask, r ristretto.Scalar
	mask.Rand()
	r.Rand()
	
	var R ristretto.Point
	R.ScalarMultBase(&r)
	
		pvKey, err := keyPair.PrivateView()
		assert.Nil(t, err)

	encryptedMask := encryptMask(mask, r, 0, *keyPair.PublicKey().PubView)

	decryptedMask := decryptMask(encryptedMask, R, 0, *pvKey)

	assert.Equal(t, decryptedMask, mask)
}
