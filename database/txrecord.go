package database

import (
	"encoding/binary"
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-wallet/transactions"
)

type txRecord struct {
	Timestamp int64
	Type      transactions.TxType
	Amount    uint64
}

type TxInRecord struct {
	txRecord
}

type TxOutRecord struct {
	txRecord
	Recipient string
}

func decodeTxIn(b []byte, t *TxInRecord) {
	decodeTxRecord(b, &t.txRecord)
}

func decodeTxOut(b []byte, t *TxOutRecord) {
	decodeTxRecord(b, &t.txRecord)
	t.Recipient = hex.EncodeToString(b[17:])
}

func decodeTxRecord(b []byte, t *txRecord) {
	t.Timestamp = int64(binary.LittleEndian.Uint64(b[0:8]))
	t.Type = transactions.TxType(b[8])
	t.Amount = binary.LittleEndian.Uint64(b[9:17])
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
