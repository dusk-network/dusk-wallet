package transactions

import (
	"bytes"
	"encoding/binary"

	"github.com/dusk-network/dusk-crypto/hash"
)

type Bid struct {
	*Timelock
	M []byte
}

func NewBid(ver uint8, netPrefix byte, fee int64, lock uint64, M []byte) (*Bid, error) {
	tx, err := NewTimelock(ver, netPrefix, fee, lock)
	if err != nil {
		return nil, err
	}

	tx.TxType = BidType
	return &Bid{
		tx,
		M,
	}, nil
}

func (b *Bid) CalculateHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := marshalBid(buf, b); err != nil {
		return nil, err
	}

	txid, err := hash.Sha3256(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return txid, nil
}

func (b *Bid) StandardTx() *Standard {
	return b.Standard
}

func (b *Bid) Type() TxType {
	return b.TxType
}

func (b *Bid) Prove() error {
	return b.prove(b.CalculateHash, false)
}

func (b *Bid) Equals(t Transaction) bool {
	other, ok := t.(*Bid)
	if !ok {
		return false
	}

	if !b.Timelock.Equals(other.Timelock) {
		return false
	}

	if !bytes.Equal(b.M, other.M) {
		return false
	}

	return true
}

func (b *Bid) LockTime() uint64 {
	return b.Lock
}

func marshalBid(b *bytes.Buffer, bid *Bid) error {
	if err := marshalTimelock(b, bid.Timelock); err != nil {
		return err
	}

	if err := binary.Write(b, binary.BigEndian, bid.M); err != nil {
		return err
	}

	return nil
}
