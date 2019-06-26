package wallet

import (
	"crypto/rand"
	"dusk-wallet/database"
	"dusk-wallet/key"
	"dusk-wallet/mlsag"
	"dusk-wallet/transactions/v3"
	"fmt"
	"math/big"

	"github.com/bwesterb/go-ristretto"
	"github.com/syndtr/goleveldb/leveldb"
)

// Number of mixins per ring. ringsize = mixin + 1
const numMixins = 7

// FetchInputs returns a slice of inputs such that Sum(Inputs)- Sum(Outputs) >= 0
// If > 0, then a change address is created for the remaining amount
type FetchInputs func(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error)

type Wallet struct {
	db          *database.DB
	netPrefix   byte
	keyPair     *key.Key
	fetchDecoys transactions.FetchDecoys
	fetchInputs FetchInputs
}

func New(netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs) (*Wallet, error) {

	// random seed
	seed := make([]byte, 64)
	_, err := rand.Read(seed)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		db:          db,
		netPrefix:   netPrefix,
		keyPair:     key.NewKeyPair(seed),
		fetchDecoys: fDecoys,
		fetchInputs: fInputs,
	}, nil
}

func (w *Wallet) NewStandardTx(fee int64) (*transactions.StandardTx, error) {
	tx, err := transactions.NewStandard(w.netPrefix, fee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (w *Wallet) NewCoinbaseTx() *transactions.CoinbaseTx {
	tx := transactions.NewCoinBaseTx(w.netPrefix)
	return tx
}

// CheckBlockSpent checks if the block has any outputs spent by this wallet
// returns the public keys of the spent outputs.
// Note: utxos are referenced via their public keys in v3 and not offset
func (w *Wallet) CheckBlockSpent(blk Block) (uint64, error) {

	var spentCount uint64

	// Standard Txs
	for _, tx := range blk.StandardTxs {
		intermediateCount, err := w.scanInputs(tx.Inputs)
		if err != nil {
			return spentCount, err
		}
		spentCount = spentCount + intermediateCount
	}

	// Stake Txs
	for _, tx := range blk.StakeTxs {
		intermediateCount, err := w.scanInputs(tx.Inputs)
		if err != nil {
			return spentCount, err
		}
		spentCount = spentCount + intermediateCount
	}

	// Bid Txs
	for _, tx := range blk.BidTxs {
		intermediateCount, err := w.scanInputs(tx.Inputs)
		if err != nil {
			return spentCount, err
		}
		spentCount = spentCount + intermediateCount
	}

	return spentCount, nil
}

func (w *Wallet) scanInputs(inputs []*transactions.Input) (uint64, error) {

	var spentCount uint64

	for _, in := range inputs {
		pubKey, err := w.db.Get(in.KeyImage.Bytes())
		if err == leveldb.ErrNotFound {
			continue
		}
		if err != nil {
			return spentCount, err
		}

		spentCount++

		err = w.db.RemoveInput(pubKey)
		if err != nil {
			return spentCount, err
		}
	}
	return spentCount, nil
}

// CheckBlockReceived checks if the block has transactions for this wallet
func (w *Wallet) CheckBlockReceived(blk Block) (uint64, error) {

	var receiveCount uint64

	// Coinbase tx
	intermediateCount, err := w.scanOutputs(false, blk.CoinbaseTx.R, blk.CoinbaseTx.Rewards)
	if err != nil {
		return 0, err
	}
	receiveCount = receiveCount + intermediateCount

	// Standard Tx
	for _, tx := range blk.StandardTxs {
		intermediateCount, err := w.scanOutputs(true, tx.R, tx.Outputs)
		if err != nil {
			return receiveCount, err
		}
		receiveCount = receiveCount + intermediateCount
	}

	// Stake Txs
	for _, tx := range blk.StakeTxs {
		intermediateCount, err := w.scanOutputs(true, tx.R, tx.Outputs)
		if err != nil {
			return receiveCount, err
		}
		receiveCount = receiveCount + intermediateCount
	}

	// bid Txs
	for _, tx := range blk.BidTxs {
		intermediateCount, err := w.scanOutputs(true, tx.R, tx.Outputs)
		if err != nil {
			return receiveCount, err
		}
		receiveCount = receiveCount + intermediateCount
	}

	return receiveCount, nil
}

func (w *Wallet) scanOutputs(valuesEncrypted bool, R ristretto.Point, outputs []*transactions.Output) (uint64, error) {

	privView, err := w.keyPair.PrivateView()
	if err != nil {
		return 0, err
	}
	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, err
	}

	var receiveCount uint64

	for i, output := range outputs {
		privKey, ok := w.keyPair.DidReceiveTx(R, output.PubKey, output.Index)
		if !ok {
			continue
		}

		receiveCount++

		var amount, mask ristretto.Scalar
		amount.Set(&output.EncryptedAmount)
		mask.Set(&output.EncryptedMask)

		if valuesEncrypted {
			amount = transactions.DecryptAmount(output.EncryptedAmount, R, uint32(i), *privView)
			mask = transactions.DecryptMask(output.EncryptedMask, R, uint32(i), *privView)
		}

		err := w.db.PutInput(privSpend.Bytes(), output.PubKey.P, amount, mask, *privKey)
		if err != nil {
			return receiveCount, err
		}

		// cache the keyImage, so we can quickly check whether our input was spent
		var pubKey ristretto.Point
		pubKey.ScalarMultBase(privKey)
		keyImage := mlsag.CalculateKeyImage(*privKey, pubKey)

		err = w.db.Put(keyImage.Bytes(), output.PubKey.P.Bytes())
		if err != nil {
			return receiveCount, err
		}
	}

	return receiveCount, nil
}

// AddInputs adds up the total outputs and fee then fetches inputs to consolidate this
func (w *Wallet) AddInputs(tx *transactions.StandardTx) error {

	totalAmount := tx.Fee.BigInt().Int64() + tx.TotalSent.BigInt().Int64()

	inputs, changeAmount, err := w.fetchInputs(w.netPrefix, w.db, totalAmount, w.keyPair)
	if err != nil {
		return err
	}
	for _, input := range inputs {
		err := tx.AddInput(input)
		if err != nil {
			return err
		}
	}

	changeAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	if err != nil {
		return err
	}

	// Convert int64 to ristretto value
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(changeAmount))

	return tx.AddOutput(*changeAddr, x)
}

func (w *Wallet) Sign(tx *transactions.StandardTx) error {

	// Assuming user has added all of the outputs

	// Fetch Inputs
	err := w.AddInputs(tx)
	if err != nil {
		fmt.Println("WE have an add input error")
		return err
	}

	// Fetch decoys
	err = tx.AddDecoys(numMixins, w.fetchDecoys)
	if err != nil {
		return err
	}

	return tx.Prove()
}

func (w *Wallet) PublicKey() key.PublicKey {
	return *w.keyPair.PublicKey()
}

// Save saves the private key information to a json file
func (w *Wallet) Save() error {
	// XXX: Have a json file
	// encrypt only the private keys with a password

	//filename can be hash of first public key
	// Ensures that files are not overwritten
	return nil
}

func LoadWallet() (*Wallet, error) {
	// XXX: Load wallet from json file
	// Will take a password from cli to un-encrypt the private keys
	return nil, nil
}
