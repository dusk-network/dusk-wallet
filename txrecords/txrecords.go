package txrecords

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/dusk-network/dusk-wallet/transactions"
)

// Direction is an enum which tells us whether a transaction is
// incoming or outgoing.
type Direction uint8

const (
	In Direction = iota
	Out
)

type TxRecord struct {
	Direction
	Timestamp int64
	transactions.TxType
	Amount       uint64
	UnlockHeight uint64
	Recipient    string
}

func New(tx transactions.Transaction, direction Direction) *TxRecord {
	return &TxRecord{
		Direction:    direction,
		Timestamp:    time.Now().Unix(),
		TxType:       tx.Type(),
		Amount:       tx.StandardTx().Outputs[0].EncryptedAmount.BigInt().Uint64(),
		UnlockHeight: tx.LockTime(),
		Recipient:    hex.EncodeToString(tx.StandardTx().Outputs[0].PubKey.P.Bytes()),
	}
}

func Encode(b *bytes.Buffer, t *TxRecord) error {
	return nil
}

func Decode(b *bytes.Buffer, t *TxRecord) error {
	return nil
}
