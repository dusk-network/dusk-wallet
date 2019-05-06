package transactions

import (
	"crypto/rand"
	"dusk-wallet/key"
	"dusk-wallet/mlsag"
	"dusk-wallet/rangeproof"
	"math/big"
	"testing"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestCommToZero(t *testing.T) {

	netPrefix := byte(12)

	fee := int64(0)
	numInputs := 2
	numOutputs := 2

	tx, err := New(netPrefix, fee)
	assert.Nil(t, err)

	var inputAmount ristretto.Scalar
	inputAmount.SetBigInt(big.NewInt(20))
	for i := 0; i < numInputs; i++ {
		input := generateInput(inputAmount)
		tx.AddInput(input)
	}

	Alice := key.NewKeyPair([]byte("this is the users seed"))
	pubAddr, err := Alice.PublicKey().PublicAddress(netPrefix)
	assert.Nil(t, err)

	var amountToSend ristretto.Scalar
	amountToSend.SetBigInt(big.NewInt(20))
	for i := 0; i < numOutputs; i++ {
		err = tx.AddOutput(*pubAddr, amountToSend)
		assert.Nil(t, err)
	}

	// Add decoys
	tx.AddDecoys(2, generateDecoys)

	err = tx.CalcCommToZero()
	assert.Nil(t, err)

	// Check MLSAG input proofs
	for _, input := range tx.Inputs {
		ok, err := input.Verify()
		assert.Nil(t, err)
		assert.True(t, ok)
	}

	// Check rangeproofs on outputs
	for _, output := range tx.Outputs {
		ok, err := rangeproof.Verify(output.RangeProof)
		assert.Nil(t, err)
		assert.True(t, ok)
	}

	// Add up pseudoCommitments
	var totalPseudo ristretto.Point
	totalPseudo.SetZero()
	for _, input := range tx.Inputs {
		totalPseudo.Add(&totalPseudo, &input.PseudoCommitment)
	}

	// Add up output commitments
	var totalOutput ristretto.Point
	totalOutput.SetZero()
	for _, output := range tx.Outputs {
		totalOutput.Add(&totalOutput, &output.Commitment)
	}

	// sum(pseudoCommitments) - sum(OutputCommitments) - fee*H= 0
	var diff, zero, feeH ristretto.Point
	zero.SetZero()

	feeH.ScalarMult(&feeH, &tx.Fee)

	diff.Sub(&totalPseudo, &totalOutput)
	diff.Sub(&diff, &feeH)

	assert.True(t, diff.Equals(&zero))
}

func generateInput(amount ristretto.Scalar) *Input {
	txid := make([]byte, 32)
	rand.Read(txid)

	var mask ristretto.Scalar
	mask.Rand()

	comm := CommitAmount(amount, mask)

	pubKey, privKey := generateKeyPair()

	input := NewInput(txid, comm, amount, mask, pubKey, privKey)
	return input
}

func generateKeyPair() (ristretto.Point, ristretto.Scalar) {
	var privKey ristretto.Scalar
	var pubKey ristretto.Point

	privKey.Rand()
	pubKey.ScalarMultBase(&privKey)

	return pubKey, privKey
}

func generateDecoys(numMixins int, numKeysPerUser int) []mlsag.PubKeys {
	var pubKeys []mlsag.PubKeys
	for i := 0; i < numMixins; i++ {
		var pubKeyVector mlsag.PubKeys
		for j := 0; j < numKeysPerUser; j++ {
			var p ristretto.Point
			p.Rand()
			pubKeyVector.AddPubKey(p)
		}
		pubKeys = append(pubKeys, pubKeyVector)
	}
	return pubKeys
}
