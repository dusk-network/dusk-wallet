package txrecords

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io/ioutil"
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
	Height    uint64
	transactions.TxType
	Amount       uint64
	UnlockHeight uint64
	Recipient    string
}

func New(tx transactions.Transaction, height uint64, direction Direction) *TxRecord {
	return &TxRecord{
		Direction:    direction,
		Timestamp:    time.Now().Unix(),
		Height:       height,
		TxType:       tx.Type(),
		Amount:       tx.StandardTx().Outputs[0].EncryptedAmount.BigInt().Uint64(),
		UnlockHeight: height + tx.LockTime(),
		Recipient:    hex.EncodeToString(tx.StandardTx().Outputs[0].PubKey.P.Bytes()),
	}
}

func Encode(b *bytes.Buffer, t *TxRecord) error {
	if err := binary.Write(b, binary.LittleEndian, t.Direction); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.Timestamp); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.Height); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.TxType); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.Amount); err != nil {
		return err
	}

	if err := binary.Write(b, binary.LittleEndian, t.UnlockHeight); err != nil {
		return err
	}

	_, err := b.Write([]byte(t.Recipient))
	return err
}

func Decode(b *bytes.Buffer, t *TxRecord) error {
	if err := binary.Read(b, binary.LittleEndian, &t.Direction); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.Timestamp); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.Height); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.TxType); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.Amount); err != nil {
		return err
	}

	if err := binary.Read(b, binary.LittleEndian, &t.UnlockHeight); err != nil {
		return err
	}

	recipientBytes, err := ioutil.ReadAll(b)
	if err != nil {
		return err
	}

	t.Recipient = string(recipientBytes)
	return nil
}
