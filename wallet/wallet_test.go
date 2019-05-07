package wallet

import (
	"dusk-wallet/key"
	"dusk-wallet/mlsag"
	"dusk-wallet/rangeproof"
	"dusk-wallet/transactions"
	"math/big"
	"math/rand"
	"testing"
	"time"

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

	sendAddr := generateSendAddr(t, netPrefix, key.NewKeyPair([]byte("this is the users seed")))

	var tenDusk ristretto.Scalar
	tenDusk.SetBigInt(big.NewInt(10))

	// Send DUSK
	numOutputs := 2
	for i := 0; i < numOutputs; i++ {
		err = tx.AddOutput(sendAddr, tenDusk)
		assert.Nil(t, err)
	}

	err = w.Sign(tx)
	assert.Nil(t, err)

	assert.True(t, len(tx.Inputs) > 0)
	assert.True(t, len(tx.Outputs) > 0)

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

	// Check receiver can spend from first two outputs
	for i := 0; i < numOutputs; i++ {
		output := tx.Outputs[i]
		ReceiversKeyPair := key.NewKeyPair([]byte("this is the users seed"))
		_, ok := ReceiversKeyPair.DidReceiveTx(tx.R, output.DestKey, output.Index)
		assert.True(t, ok)
	}
}

func TestReceivedTx(t *testing.T) {
	netPrefix := byte(1)
	fee := int64(0)

	w, err := New(netPrefix, generateDecoys, fetchInputs)
	assert.Nil(t, err)

	tx, err := w.NewStealthTx(fee)
	assert.Nil(t, err)

	var tenDusk ristretto.Scalar
	tenDusk.SetBigInt(big.NewInt(10))

	sendersAddr := generateSendAddr(t, netPrefix, w.keyPair)
	assert.Nil(t, err)

	err = tx.AddOutput(sendersAddr, tenDusk)
	assert.Nil(t, err)

	err = w.Sign(tx)
	assert.Nil(t, err)

	for _, output := range tx.Outputs {
		_, ok := w.keyPair.DidReceiveTx(tx.R, output.DestKey, output.Index)
		assert.True(t, ok)
	}

	var destKeys []ristretto.Point
	for _, output := range tx.Outputs {
		destKeys = append(destKeys, output.DestKey.P)
	}
	assert.False(t, hasDuplicates(destKeys))
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
	numInputs := 7

	addresses, R := generateOutputAddress(key, netPrefix, numInputs)

	rand.Seed(time.Now().Unix())
	randAmount := rand.Int63n(totalAmount) + int64(totalAmount)/int64(numInputs) + 1 // [totalAmount/4 + 1, totalAmount*4]
	remainder := (randAmount * int64(numInputs)) - totalAmount
	if remainder < 0 {
		remainder = 0
	}

	for index, addr := range addresses {
		txid := []byte{2}
		var amount, mask ristretto.Scalar
		amount.SetBigInt(big.NewInt(randAmount))
		mask.Rand()
		commitment := transactions.CommitAmount(amount, mask)

		// Fetch the privKey for each addresses
		privKey, _ := key.DidReceiveTx(R, *addr, uint32(index))

		input := transactions.NewInput(txid, commitment, amount, mask, addr.P, *privKey)
		inputs = append(inputs, input)
	}

	return inputs, remainder, nil
}

func generateSendAddr(t *testing.T, netPrefix byte, randKeyPair *key.Key) key.PublicAddress {
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

// https://www.dotnetperls.com/duplicates-go
func hasDuplicates(elements []ristretto.Point) bool {
	encountered := map[ristretto.Point]bool{}

	for v := range elements {
		if encountered[elements[v]] == true {
			return true
		}
		encountered[elements[v]] = true
	}
	return false
}
