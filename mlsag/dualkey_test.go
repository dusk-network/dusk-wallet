package mlsag

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestDualKey(t *testing.T) {

	dk := generateRandDualKeyProof(20)

	sig, keyImage, err := dk.Prove()
	sig.Verify([]ristretto.Point{keyImage})
	assert.Nil(t, err)
}

func generateRandDualKeyProof(numUsers int) *DualKey {
	proof := NewDualKey()

	numDecoys := numUsers - 1
	numKeys := 2

	// Generate and add decoys to proof
	matrixPubKeys := generateDecoys(numDecoys, numKeys)
	for i := 0; i < len(matrixPubKeys); i++ {
		pubKeys := matrixPubKeys[i]
		proof.AddDecoy(pubKeys)
	}

	// Generate and add private keys to proof
	var primaryKey, commToZero ristretto.Scalar
	primaryKey.Rand()
	commToZero.Rand()
	proof.SetPrimaryKey(primaryKey)
	proof.SetCommToZero(commToZero)

	proof.msg = []byte("hello world")

	return proof
}
