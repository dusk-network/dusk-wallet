package database

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/dusk-network/dusk-wallet/transactions"

	"github.com/bwesterb/go-ristretto"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type DB struct {
	storage *leveldb.DB
}

var (
	inputPrefix        = []byte("input")
	walletHeightPrefix = []byte("syncedHeight")

	writeOptions = &opt.WriteOptions{NoWriteMerge: false, Sync: true}
)

func New(path string) (*DB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, fmt.Errorf("wallet cannot be used without database %s", err.Error())
	}
	return &DB{storage: db}, nil
}

func (db *DB) Put(key, value []byte) error {
	return db.storage.Put(key, value, nil)
}

func (db *DB) PutInput(encryptionKey []byte, pubkey ristretto.Point, amount, mask, privKey ristretto.Scalar, lockHeight uint64) error {

	buf := &bytes.Buffer{}
	idb := &inputDB{
		amount:     amount,
		mask:       mask,
		privKey:    privKey,
		lockHeight: lockHeight,
	}

	if err := idb.Encode(buf); err != nil {
		return err
	}

	encryptedBytes, err := encrypt(buf.Bytes(), encryptionKey)
	if err != nil {
		return err
	}

	key := append(inputPrefix, pubkey.Bytes()...)

	return db.Put(key, encryptedBytes)
}

func (db *DB) RemoveInput(pubkey []byte, keyImage []byte) error {
	key := append(inputPrefix, pubkey...)

	b := new(leveldb.Batch)
	b.Delete(key)
	b.Delete(keyImage)

	return db.storage.Write(b, writeOptions)
}

func (db *DB) FetchInputs(decryptionKey []byte, amount int64) ([]*transactions.Input, int64, error) {

	var inputs []*inputDB

	var totalAmount = amount

	iter := db.storage.NewIterator(util.BytesPrefix(inputPrefix), nil)
	defer iter.Release()
	for iter.Next() {
		val := iter.Value()

		encryptedBytes := make([]byte, len(val))
		copy(encryptedBytes[:], val)

		decryptedBytes, err := decrypt(encryptedBytes, decryptionKey)
		if err != nil {
			return nil, 0, err
		}
		idb := &inputDB{}

		buf := bytes.NewBuffer(decryptedBytes)
		err = idb.Decode(buf)
		if err != nil {
			return nil, 0, err
		}

		// Only add unlocked inputs
		if idb.lockHeight == 0 {
			inputs = append(inputs, idb)

			// Check if we need more inputs
			totalAmount = totalAmount - idb.amount.BigInt().Int64()
			if totalAmount <= 0 {
				break
			}
		}
	}

	if totalAmount > 0 {
		return nil, 0, errors.New("accumulated value of all of your inputs do not account for the total amount inputted")
	}

	err := iter.Error()
	if err != nil {
		return nil, 0, err
	}

	var changeAmount int64
	if totalAmount < 0 {
		changeAmount = -totalAmount
	}

	// convert inputDb to transaction input
	var tInputs []*transactions.Input
	for _, input := range inputs {
		tInputs = append(tInputs, transactions.NewInput(input.amount, input.mask, input.privKey))
	}

	return tInputs, changeAmount, nil
}

func (db *DB) FetchBalance(decryptionKey []byte) (uint64, error) {
	var balance ristretto.Scalar
	balance.SetZero()

	iter := db.storage.NewIterator(util.BytesPrefix(inputPrefix), nil)
	defer iter.Release()
	for iter.Next() {
		val := iter.Value()

		encryptedBytes := make([]byte, len(val))
		copy(encryptedBytes[:], val)

		decryptedBytes, err := decrypt(encryptedBytes, decryptionKey)
		if err != nil {
			return 0, err
		}
		idb := &inputDB{}

		buf := bytes.NewBuffer(decryptedBytes)
		err = idb.Decode(buf)
		if err != nil {
			return 0, err
		}

		balance.Add(&balance, &idb.amount)

	}

	err := iter.Error()
	if err != nil {
		return 0, err
	}

	return balance.BigInt().Uint64(), nil
}

// UpdateLockedInputs will set the lockheight for an input to 0 if the
// given `height` is greater or equal than the input lockheight,
// signifying that this input is unlocked.
func (db *DB) UpdateLockedInputs(decryptionKey []byte, height uint64) error {
	iter := db.storage.NewIterator(util.BytesPrefix(inputPrefix), nil)
	defer iter.Release()
	for iter.Next() {
		val := iter.Value()

		decryptedBytes, err := decrypt(val, decryptionKey)
		if err != nil {
			return err
		}
		idb := &inputDB{}

		buf := bytes.NewBuffer(decryptedBytes)
		err = idb.Decode(buf)
		if err != nil {
			return err
		}

		if idb.lockHeight != 0 && idb.lockHeight <= height {
			idb.lockHeight = 0
			// Overwrite input
			buf := new(bytes.Buffer)
			if err := idb.Encode(buf); err != nil {
				return err
			}

			encryptedBytes, err := encrypt(buf.Bytes(), decryptionKey)
			if err != nil {
				return err
			}

			db.Put(iter.Key(), encryptedBytes)
		}
	}

	return iter.Error()
}

func (db *DB) GetWalletHeight() (uint64, error) {
	heightBytes, err := db.storage.Get(walletHeightPrefix, nil)
	if err != nil {
		return 0, err
	}

	height := binary.LittleEndian.Uint64(heightBytes)
	return height, nil
}

func (db *DB) UpdateWalletHeight(newHeight uint64) error {
	heightBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightBytes, newHeight)
	return db.Put(walletHeightPrefix, heightBytes)
}

func (db *DB) Get(key []byte) ([]byte, error) {
	return db.storage.Get(key, nil)
}

func (db *DB) Delete(key []byte) error {
	return db.storage.Delete(key, nil)
}

func (db *DB) Close() error {
	return db.storage.Close()
}
