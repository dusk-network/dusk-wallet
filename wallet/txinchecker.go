package wallet

import "github.com/dusk-network/dusk-wallet/transactions"

type keyImage []byte

//TxInChecker contains the necessary information to
// deduce whether a user has spent a tx. This is just the keyImage.
type TxInChecker struct {
	keyImages []keyImage
}

func NewTxInChecker(txs []transactions.Transaction) []TxInChecker {
	txcheckers := make([]TxInChecker, 0, len(txs))

	for _, tx := range txs {
		keyImages := make([]keyImage, 0)
		for _, input := range tx.StandardTx().Inputs {
			keyImages = append(keyImages, input.KeyImage.Bytes())
		}
		txcheckers = append(txcheckers, TxInChecker{keyImages})
	}
	return txcheckers
}
