package wallet

import (
	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/dusk-network/dusk-wallet/transactions"
	wiretx "github.com/dusk-network/dusk-wallet/transactions"
)

// CheckWireBlockReceived checks if the wire block has transactions for this wallet
// Returns the number of tx's that the reciever recieved funds in
func (w *Wallet) CheckWireBlockReceived(blk block.Block) (uint64, error) {
	privView, err := w.keyPair.PrivateView()
	if err != nil {
		return 0, err
	}

	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, err
	}

	var totalReceivedCount uint64

	for _, tx := range blk.Txs {
		for i, output := range tx.StandardTx().Outputs {
			privKey, ok := w.keyPair.DidReceiveTx(tx.StandardTx().R, output.PubKey, uint32(i))
			if !ok {
				continue
			}

			totalReceivedCount++

			w.writeOutputToDatabase(*output, privView, privSpend, *privKey, tx, i)
			w.writeKeyImageToDatabase(*output, *privKey)
		}
	}

	return totalReceivedCount, nil
}

func (w *Wallet) writeOutputToDatabase(output transactions.Output, privView *key.PrivateView, privSpend *key.PrivateSpend, privKey ristretto.Scalar, tx transactions.Transaction, i int) error {
	var amount, mask ristretto.Scalar
	amount.Set(&output.EncryptedAmount)
	mask.Set(&output.EncryptedMask)

	if shouldEncryptValues(tx) {
		amount = transactions.DecryptAmount(output.EncryptedAmount, tx.StandardTx().R, uint32(i), *privView)
		mask = transactions.DecryptMask(output.EncryptedMask, tx.StandardTx().R, uint32(i), *privView)
	}

	return w.db.PutInput(privSpend.Bytes(), output.PubKey.P, amount, mask, privKey, getLockTime(tx))
}

func (w *Wallet) writeKeyImageToDatabase(output transactions.Output, privKey ristretto.Scalar) error {
	// cache the keyImage, so we can quickly check whether our input was spent
	var pubKey ristretto.Point
	pubKey.ScalarMultBase(&privKey)
	keyImage := mlsag.CalculateKeyImage(privKey, pubKey)
	return w.db.Put(keyImage.Bytes(), output.PubKey.P.Bytes())
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

// Not all transaction types have a Lock field, so this function will
// discern the transaction passed based on it's type, and either return
// the associated Lock field value, or 0.
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
