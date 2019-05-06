package mlsag

import (
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestKeyImages(t *testing.T) {
	for i := 1; i < 20; i++ {
		proof := Proof{}
		proof.AddSecret(generatePrivKeys(i))
		KeyImages := proof.calculateKeyImages()
		assert.Equal(t, i, len(KeyImages))
	}
}
func TestGenNonces(t *testing.T) {
	for i := 1; i < 20; i++ {
		nonces := generateNonces(i)
		assert.Equal(t, i, len(nonces))
	}
}

func TestShuffleSet(t *testing.T) {
	numUsers := 3
	numKeys := 1

	proof := generateRandProof(numUsers, numKeys)
	err := proof.shuffleSet()
	assert.Equal(t, nil, err)

	signersPubKeys := proof.pubKeysMatrix[proof.index]
	assert.Equal(t, false, signersPubKeys.decoy)
}

func TestMLSAGProveVerify(t *testing.T) {

	numUsers := 10
	numKeys := 34

	proof := generateRandProof(numUsers, numKeys)

	sig, err := proof.Prove()
	assert.Equal(t, nil, err)

	assert.Equal(t, numKeys, len(sig.KeyImages))
	assert.Equal(t, numUsers, len(sig.PubKeys))
	assert.Equal(t, proof.calculateKeyImages(), sig.KeyImages)
	assert.Equal(t, proof.pubKeysMatrix, sig.PubKeys)
	assert.Equal(t, proof.msg, sig.Msg)

	ok, err := sig.Verify()
	assert.Equal(t, nil, err)
	assert.True(t, ok)
}

func TestMLSAGBadSig(t *testing.T) {

	numUsers := 12
	numKeys := 10

	proof := generateRandProof(numUsers, numKeys)

	sig, err := proof.Prove()
	assert.Equal(t, nil, err)

	sig.Msg = []byte("something random")

	ok, err := sig.Verify()
	assert.NotNil(t, err)
	assert.False(t, ok)
}

func generateDecoy(n int) PubKeys {
	var pKeys PubKeys

	for i := 0; i < n; i++ {
		var randPoint ristretto.Point
		randPoint.Rand()
		pKeys.AddPubKey(randPoint)
	}

	return pKeys
}

func generateDecoys(m int, n int) []PubKeys {
	var matrixPubKeys []PubKeys
	for i := 0; i < m; i++ {
		matrixPubKeys = append(matrixPubKeys, generateDecoy(n))
	}
	return matrixPubKeys
}

func generatePrivKeys(m int) PrivKeys {
	var privKeys PrivKeys
	for i := 0; i < m; i++ {
		var p ristretto.Scalar
		p.Rand()
		privKeys.AddPrivateKey(p)
	}
	return privKeys
}

func generateRandProof(numUsers, numKeys int) *Proof {
	proof := &Proof{}

	numDecoys := numUsers - 1

	// Generate and add decoys to proof
	matrixPubKeys := generateDecoys(numDecoys, numKeys)
	for i := 0; i < len(matrixPubKeys); i++ {
		pubKeys := matrixPubKeys[i]
		proof.AddDecoy(pubKeys)
	}

	// Generate and add private keys to proof
	privKeys := generatePrivKeys(numKeys)
	proof.AddSecret(privKeys)

	proof.msg = []byte("hello world")

	return proof
}
