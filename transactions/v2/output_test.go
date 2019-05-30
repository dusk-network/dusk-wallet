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
