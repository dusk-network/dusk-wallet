package database

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

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
	txInPrefix         = []byte("txIn")
	txOutPrefix        = []byte("txOut")

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

func (db *DB) PutTxIn(tx transactions.Transaction) error {
	// Schema
	//
	// key: txInPrefix
	// value: timestamp(unix) + type + amount

	// 8 (timestamp) + 1 (type) + 8 (amount)
	value := make([]byte, 17)

	putCommonFields(value, tx)

	return db.Put(txInPrefix, value)
}

func (db *DB) PutTxOut(tx transactions.Transaction) error {
	// Schema
	//
	// key: txOutPrefix
	// value: timestamp(unix) + type + amount + address

	// 8 (timestamp) + 1 (type) + 8 (amount)
	value := make([]byte, 17)

	putCommonFields(value, tx)

	// Address
	value = append(value, tx.StandardTx().Outputs[0].PubKey.P.Bytes()...)

	return db.Put(txOutPrefix, value)
}

// Common fields for tx records, whether they are in or out.
func putCommonFields(value []byte, tx transactions.Transaction) {
	// Timestamp
	time := time.Now().Unix()
	binary.LittleEndian.PutUint64(value[0:8], uint64(time))

	// Type
	value[8] = byte(tx.Type())

	// Amount
	binary.LittleEndian.PutUint64(value[9:17], tx.StandardTx().Outputs[0].EncryptedAmount.BigInt().Uint64())
}

func (db *DB) PutInput(encryptionKey []byte, pubkey ristretto.Point, amount, mask, privKey ristretto.Scalar, unlockHeight uint64) error {

	buf := &bytes.Buffer{}
	idb := &inputDB{
		amount:       amount,
		mask:         mask,
		privKey:      privKey,
		unlockHeight: unlockHeight,
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
		if idb.unlockHeight == 0 {
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

func (db *DB) FetchBalance(decryptionKey []byte) (uint64, uint64, error) {
	var unlockedBalance ristretto.Scalar
	unlockedBalance.SetZero()
	var lockedBalance ristretto.Scalar
	lockedBalance.SetZero()

	iter := db.storage.NewIterator(util.BytesPrefix(inputPrefix), nil)
	defer iter.Release()
	for iter.Next() {
		val := iter.Value()

		encryptedBytes := make([]byte, len(val))
		copy(encryptedBytes[:], val)

		decryptedBytes, err := decrypt(encryptedBytes, decryptionKey)
		if err != nil {
			return 0, 0, err
		}
		idb := &inputDB{}

		buf := bytes.NewBuffer(decryptedBytes)
		err = idb.Decode(buf)
		if err != nil {
			return 0, 0, err
		}

		if idb.unlockHeight == 0 {
			unlockedBalance.Add(&unlockedBalance, &idb.amount)
			continue
		}

		lockedBalance.Add(&lockedBalance, &idb.amount)
	}

	err := iter.Error()
	if err != nil {
		return 0, 0, err
	}

	return unlockedBalance.BigInt().Uint64(), lockedBalance.BigInt().Uint64(), nil
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

		if idb.unlockHeight != 0 && idb.unlockHeight <= height {
			idb.unlockHeight = 0
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
