package wallet

import (
	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/transactions"
	wiretx "github.com/dusk-network/dusk-wallet/transactions"
)

// TxOutChecker holds all of the necessary data
// in order to check if an output was sent to a specified user
type TxOutChecker struct {
	encryptedValues bool
	R               ristretto.Point
	Outputs         transactions.Outputs
	LockTime        uint64
}

func NewTxOutChecker(blk block.Block) []TxOutChecker {
	txcheckers := make([]TxOutChecker, 0, len(blk.Txs))

	for _, tx := range blk.Txs {
		txchecker := TxOutChecker{
			encryptedValues: shouldEncryptValues(tx),
		}

		var RBytes [32]byte
		txR := tx.StandardTx().R
		copy(RBytes[:], txR.Bytes()[:])
		var R ristretto.Point
		R.SetBytes(&RBytes)
		txchecker.R = R

		txchecker.Outputs = tx.StandardTx().Outputs
		txcheckers = append(txcheckers, txchecker)

		txchecker.LockTime = getLockTime(tx)
	}
	return txcheckers
}

func shouldEncryptValues(tx wiretx.Transaction) bool {
	switch tx.Type() {
	case wiretx.StandardType:
		return true
	case wiretx.TimelockType:
		return true
	case wiretx.BidType:
		return false
	case wiretx.StakeType:
		return false
	case wiretx.CoinbaseType:
		return false
	default:
		return true
	}
}

func getLockTime(tx wiretx.Transaction) uint64 {
	switch tx.Type() {
	case wiretx.StandardType, wiretx.CoinbaseType:
		return 0
	case wiretx.BidType:
		bid := tx.(*wiretx.Bid)
		return bid.Lock
	case wiretx.StakeType:
		stake := tx.(*wiretx.Stake)
		return stake.Lock
	case wiretx.TimelockType:
		tl := tx.(*wiretx.Timelock)
		return tl.Lock
	default:
		return 0
	}
}
