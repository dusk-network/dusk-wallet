package database

import (
	"bytes"
	"os"
	"testing"

	"github.com/bwesterb/go-ristretto"
	"github.com/stretchr/testify/assert"
	"github.com/syndtr/goleveldb/leveldb"
)

func TestPutGet(t *testing.T) {

	path := "mainnet"

	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	// Put
	key := []byte("hello")
	value := []byte("world")
	err = db.Put(key, value)
	assert.Nil(t, err)

	// Close and re-open database
	err = db.Close()
	assert.Nil(t, err)
	db, err = New(path)
	assert.Nil(t, err)

	// Get
	val, err := db.Get(key)
	assert.Nil(t, err)
	assert.True(t, bytes.Equal(val, value))

	// Delete
	err = db.Delete(key)
	assert.Nil(t, err)

	// Get after delete
	val, err = db.Get(key)
	assert.Equal(t, leveldb.ErrNotFound, err)
	assert.True(t, bytes.Equal(val, []byte{}))
}

func TestUnlockInputs(t *testing.T) {
	path := "mainnet"

	// New
	db, err := New(path)
	assert.Nil(t, err)

	// Make sure to delete this dir after test
	defer os.RemoveAll(path)

	input := randInput()
	// This input unlocks at height 1000
	input.unlockHeight = 1000

	// Put it in the DB
	var pubKey ristretto.Point
	pubKey.Rand()
	assert.NoError(t, db.PutInput([]byte{0}, pubKey, input.amount, input.mask, input.privKey, input.unlockHeight))

	// Fetch it and ensure the unlock height is set
	key := append(inputPrefix, pubKey.Bytes()...)
	value, err := db.Get(key)
	assert.NoError(t, err)

	decoded := &inputDB{}
	decoded.Decode(bytes.NewBuffer(value))

	assert.Equal(t, uint64(1000), decoded.unlockHeight)

	// Now run UpdateLockedInputs
	assert.NoError(t, db.UpdateLockedInputs([]byte{0}, 1000))

	value, err = db.Get(key)
	assert.NoError(t, err)

	decoded = &inputDB{}
	decoded.Decode(bytes.NewBuffer(value))

	assert.Equal(t, uint64(0), decoded.unlockHeight)
}

func randInput() *inputDB {
	var amount, mask, privKey ristretto.Scalar
	amount.Rand()
	mask.Rand()
	privKey.Rand()
	idb := &inputDB{
		amount:  amount,
		mask:    mask,
		privKey: privKey,
	}

	return idb
}
