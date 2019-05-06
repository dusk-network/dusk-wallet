package wallet

import (
	"dusk-wallet/key"
	"dusk-wallet/mlsag"
	"dusk-wallet/rangeproof"
	"dusk-wallet/transactions"
	"math/big"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
)

func TestWallet(t *testing.T) {

	netPrefix := byte(1)
	fee := int64(20)

	w, err := New(netPrefix, generateDecoys, fetchInputs)
	assert.Nil(t, err)

	tx, err := w.NewStealthTx(fee)
	assert.Nil(t, err)

	sendAddr := generateSendAddr(t, netPrefix)

	var tenDusk ristretto.Scalar
	tenDusk.SetBigInt(big.NewInt(10))

	// Send 20 DUSK
	err = tx.AddOutput(sendAddr, tenDusk)
	assert.Nil(t, err)
	err = tx.AddOutput(sendAddr, tenDusk)
	assert.Nil(t, err)

	err = w.Sign(tx)
	assert.Nil(t, err)

	for _, input := range tx.Inputs {
		ok, err := input.Verify()
		assert.True(t, ok)
		assert.Nil(t, err)
	}
	for _, output := range tx.Outputs {
		ok, err := rangeproof.Verify(output.RangeProof)
		assert.True(t, ok)
		assert.Nil(t, err)
	}
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

func fetchInputs(netPrefix byte, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {

	// This function shoud store the inputs in a database
	// Upon calling fetchInputs, we use the keyPair to get the privateKey from the
	// one time pubkey

	var inputs []*transactions.Input
	numInputs := 5

	addresses, R := generateOutputAddress(key, netPrefix, numInputs)

	each, remainder := splitAmount(totalAmount, int64(numInputs))

	for index, addr := range addresses {
		txid := []byte{2}
		var amount, mask ristretto.Scalar
		amount.SetBigInt(big.NewInt(each))
		mask.Rand()
		commitment := transactions.CommitAmount(amount, mask)

		// Fetch the privKey for each addresses
		privKey, _ := key.DidReceiveTx(R, *addr, uint32(index))

		input := transactions.NewInput(txid, commitment, amount, mask, addr.P, *privKey)
		inputs = append(inputs, input)
	}

	return inputs, remainder, nil
}

func generateSendAddr(t *testing.T, netPrefix byte) key.PublicAddress {
	randKeyPair := key.NewKeyPair([]byte("this is the users seed"))
	pubAddr, err := randKeyPair.PublicKey().PublicAddress(netPrefix)
	assert.Nil(t, err)
	return *pubAddr
}

func generateOutputAddress(keyPair *key.Key, netPrefix byte, num int) ([]*key.StealthAddress, ristretto.Point) {
	var res []*key.StealthAddress

	sendersPubKey := keyPair.PublicKey()

	var r ristretto.Scalar
	r.Rand()

	var R ristretto.Point
	R.ScalarMultBase(&r)

	for i := 0; i < num; i++ {
		stealthAddr := sendersPubKey.StealthAddress(r, uint32(i))
		res = append(res, stealthAddr)
	}
	return res, R
}

// Splits a number into separate buckets
func splitAmount(totalAmount, totalBuckets int64) (int64, int64) {
	remainder := totalAmount % totalBuckets
	each := totalAmount / totalBuckets
	return each, remainder
}
