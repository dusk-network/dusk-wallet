package wallet

import (
	"github.com/dusk-network/dusk-wallet/transactions/v3"
	"encoding/binary"
	"io"
)

type Block struct {
	CoinbaseTx  transactions.CoinbaseTx
	StandardTxs []transactions.StandardTx
	StakeTxs    []transactions.StakeTx
	BidTxs      []transactions.BidTx
	TimeTxs     []transactions.TimelockTx
}

func (b *Block) AddCoinbaseTx(tx transactions.CoinbaseTx) {
	b.CoinbaseTx = tx
}

func (b *Block) AddStandardTx(tx transactions.StandardTx) {
	b.StandardTxs = append(b.StandardTxs, tx)
}

func (b *Block) AddBidTx(tx transactions.BidTx) {
	b.BidTxs = append(b.BidTxs, tx)
}

func (b *Block) AddStakeTx(tx transactions.StakeTx) {
	b.StakeTxs = append(b.StakeTxs, tx)
}

func (b *Block) AddTimeLockTx(tx transactions.TimelockTx) {
	b.TimeTxs = append(b.TimeTxs, tx)
}

func (b *Block) Encode(w io.Writer) error {

	// Encode Coinbase Tx
	err := b.CoinbaseTx.Encode(w)
	if err != nil {
		return err
	}

	// Encode Standard Txs
	lenStandard := uint16(len(b.StandardTxs))
	err = binary.Write(w, binary.BigEndian, lenStandard)
	if err != nil {
		return err
	}
	for _, tx := range b.StandardTxs {
		err := tx.Encode(w)
		if err != nil {
			return err
		}
	}

	// Encode Stake Txs
	lenStake := uint16(len(b.StakeTxs))
	err = binary.Write(w, binary.BigEndian, lenStake)
	if err != nil {
		return err
	}
	for _, tx := range b.StakeTxs {
		err := tx.Encode(w)
		if err != nil {
			return err
		}
	}

	// Encode Bid Txs
	lenBid := uint16(len(b.BidTxs))
	err = binary.Write(w, binary.BigEndian, lenBid)
	if err != nil {
		return err
	}
	for _, tx := range b.BidTxs {
		err := tx.Encode(w)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Block) Decode(r io.Reader) error {

	err := b.CoinbaseTx.Decode(r)
	if err != nil {
		return err
	}

	var lenStandard uint16
	err = binary.Read(r, binary.BigEndian, &lenStandard)
	if err != nil {
		return err
	}
	for i := uint16(0); i < lenStandard; i++ {
		tx := transactions.StandardTx{}
		err := tx.Decode(r)
		if err != nil {
			return err
		}
		b.StandardTxs = append(b.StandardTxs, tx)
	}

	var lenStake uint16
	err = binary.Read(r, binary.BigEndian, &lenStake)
	if err != nil {
		return err
	}
	for i := uint16(0); i < lenStake; i++ {
		tx := transactions.StakeTx{}
		err := tx.Decode(r)
		if err != nil {
			return err
		}
		b.StakeTxs = append(b.StakeTxs, tx)
	}

	var lenBid uint16
	err = binary.Read(r, binary.BigEndian, &lenBid)
	if err != nil {
		return err
	}
	for i := uint16(0); i < lenBid; i++ {
		tx := transactions.BidTx{}
		err := tx.Decode(r)
		if err != nil {
			return err
		}
		b.BidTxs = append(b.BidTxs, tx)
	}
	return nil
}
