package wallet

import (
	"bytes"
	"dusk-wallet/key"
	"dusk-wallet/rangeproof"
	"dusk-wallet/rangeproof/pedersen"
	"dusk-wallet/transactions/v2"
	"fmt"
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
	numOutputs := 3
	for i := 0; i < numOutputs; i++ {
		err = tx.AddOutput(sendAddr, tenDusk)
		assert.Nil(t, err)
	}

	err = w.Sign(tx)
	assert.Nil(t, err)

	assert.True(t, len(tx.Inputs) > 0)
	assert.True(t, len(tx.Outputs) > 0)

	ok, err := rangeproof.Verify(tx.RangeProof)
	assert.True(t, ok)
	assert.Nil(t, err)

	// Check receiver can spend from first two outputs
	for i := 0; i < numOutputs; i++ {
		output := tx.Outputs[i]
		ReceiversKeyPair := key.NewKeyPair([]byte("this is the users seed"))
		_, ok := ReceiversKeyPair.DidReceiveTx(tx.R, output.PubKey, output.Index)
		assert.True(t, ok)
	}

	baseTx, err := tx.Encode()
	assert.Nil(t, err)

	decodedRP := &rangeproof.Proof{}
	buf := bytes.NewBuffer(baseTx.RangeProof)
	err = decodedRP.Decode(buf, false)
	assert.Nil(t, err)

	// Get all commitments from the transaction for the rangeproof
	var commitments []pedersen.Commitment
	for i := range baseTx.Outputs {
		c := sliceToPoint(t, baseTx.Outputs[i].Commitment)
		commitments = append(commitments,
			pedersen.Commitment{
				Value: c,
			})
	}
	decodedRP.V = commitments
	assert.Equal(t, len(baseTx.Outputs), len(decodedRP.V))

	// Verify rangeproof
	ok, err = rangeproof.Verify(*decodedRP)
	assert.Nil(t, err)
	assert.True(t, ok)

	// Verify it all balances out
	var sumIn ristretto.Point
	sumIn.SetZero()
	for i := range baseTx.Inputs {
		pseudoOut := sliceToPoint(t, baseTx.Inputs[i].PseudoCommitment)
		sumIn.Add(&sumIn, &pseudoOut)
	}
	var sumOut ristretto.Point
	sumOut.SetZero()
	for i := range baseTx.Outputs {
		comm := sliceToPoint(t, baseTx.Outputs[i].Commitment)
		sumOut.Add(&sumOut, &comm)
	}
	// SumIn - SumOut - Fee
	sumIn.Sub(&sumIn, &sumOut)
	var f ristretto.Scalar
	f.SetBigInt(big.NewInt(fee))
	var zero ristretto.Scalar
	zero.SetZero()
	Fee := transactions.CommitAmount(f, zero)
	sumIn.Sub(&sumIn, &Fee)

	var zeroPoint ristretto.Point

	assert.True(t, sumIn.Equals(&zeroPoint))
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
		_, ok := w.keyPair.DidReceiveTx(tx.R, output.PubKey, output.Index)
		assert.True(t, ok)
	}

	var destKeys []ristretto.Point
	for _, output := range tx.Outputs {
		destKeys = append(destKeys, output.PubKey.P)
	}
	assert.False(t, hasDuplicates(destKeys))
}

func generateDecoys(numMixins int) transactions.Decoys {
	var decoys transactions.Decoys
	var commitment, pubkey ristretto.Point

	for i := 0; i < numMixins; i++ {
		commitment.Rand()
		pubkey.Rand()
		offset := pubkey.Bytes()[:5]

		decoy := transactions.Decoy{
			Commitment: commitment,
			PubKey:     pubkey,
			Offset:     offset,
		}

		decoys = append(decoys, decoy)
	}

	return decoys
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
		var amount, mask ristretto.Scalar
		fmt.Println(randAmount)
		amount.SetBigInt(big.NewInt(randAmount))
		mask.Rand()

		// Fetch the privKey for each addresses
		privKey, _ := key.DidReceiveTx(R, *addr, uint32(index))

		input := transactions.NewInput(amount, mask, *privKey)
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

func sliceToPoint(t *testing.T, b []byte) ristretto.Point {
	if len(b) != 32 {
		t.Fatal("slice to point must be given a 32 byte slice")
	}
	var c ristretto.Point
	var byts [32]byte
	copy(byts[:], b)
	c.SetBytes(&byts)
	return c
}
