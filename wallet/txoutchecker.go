package wallet

import (
	"github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-crypto/mlsag"
	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/dusk-network/dusk-wallet/transactions"
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
		var didReceiveFunds bool
		for i, output := range tx.StandardTx().Outputs {
			privKey, ok := w.keyPair.DidReceiveTx(tx.StandardTx().R, output.PubKey, uint32(i))
			if !ok {
				continue
			}

			didReceiveFunds = true

			if err := w.writeOutputToDatabase(*output, privView, privSpend, *privKey, tx, i, blk.Header.Height); err != nil {
				return 0, err
			}

			if err := w.writeKeyImageToDatabase(*output, *privKey); err != nil {
				return 0, err
			}
		}

		if didReceiveFunds {
			totalReceivedCount++
			_ = w.db.PutTxIn(tx)
		}
	}

	return totalReceivedCount, nil
}

func (w *Wallet) writeOutputToDatabase(output transactions.Output, privView *key.PrivateView, privSpend *key.PrivateSpend, privKey ristretto.Scalar, tx transactions.Transaction, i int, blockHeight uint64) error {
	var amount, mask ristretto.Scalar
	amount.Set(&output.EncryptedAmount)
	mask.Set(&output.EncryptedMask)

	if shouldEncryptValues(tx) {
		amount = transactions.DecryptAmount(output.EncryptedAmount, tx.StandardTx().R, uint32(i), *privView)
		mask = transactions.DecryptMask(output.EncryptedMask, tx.StandardTx().R, uint32(i), *privView)
	}

	// Only the first output of a tx is locked, to avoid locking up
	// a change output.
	if i == 0 {
		return w.db.PutInput(privSpend.Bytes(), output.PubKey.P, amount, mask, privKey, tx.LockTime()+blockHeight)
	}

	return w.db.PutInput(privSpend.Bytes(), output.PubKey.P, amount, mask, privKey, 0)
}

func (w *Wallet) writeKeyImageToDatabase(output transactions.Output, privKey ristretto.Scalar) error {
	// cache the keyImage, so we can quickly check whether our input was spent
	var pubKey ristretto.Point
	pubKey.ScalarMultBase(&privKey)
	keyImage := mlsag.CalculateKeyImage(privKey, pubKey)
	return w.db.Put(keyImage.Bytes(), output.PubKey.P.Bytes())
}

func shouldEncryptValues(tx transactions.Transaction) bool {
	switch tx.Type() {
	case transactions.StandardType:
		return true
	case transactions.TimelockType:
		return true
	case transactions.BidType:
		return false
	case transactions.StakeType:
		return false
	case transactions.CoinbaseType:
		return false
	default:
		return true
	}
}
