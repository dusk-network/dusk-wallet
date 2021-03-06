package wallet

import (
	"errors"
	"fmt"
	"io"

	"github.com/dusk-network/dusk-wallet/v2/block"
	"github.com/dusk-network/dusk-wallet/v2/database"
	"github.com/dusk-network/dusk-wallet/v2/key"
	"github.com/dusk-network/dusk-wallet/v2/transactions"
	"github.com/dusk-network/dusk-wallet/v2/txrecords"

	"github.com/syndtr/goleveldb/leveldb"
)

// Number of mixins per ring. ringsize = mixin + 1
const numMixins = 7

// DUSK is one whole unit of DUSK.
const DUSK = uint64(100000000)

var ErrSeedFileExists = fmt.Errorf("wallet seed file already exists")

// FetchInputs returns a slice of inputs such that Sum(Inputs)- Sum(Outputs) >= 0
// If > 0, then a change address is created for the remaining amount
type FetchInputs func(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error)

type Wallet struct {
	db        *database.DB
	netPrefix byte

	keyPair       *key.Key
	consensusKeys *key.ConsensusKeys

	fetchDecoys transactions.FetchDecoys
	fetchInputs FetchInputs
}

type SignableTx interface {
	AddDecoys(numMixins int, f transactions.FetchDecoys) error
	Prove() error
	StandardTx() *transactions.Standard
}

func New(Read func(buf []byte) (n int, err error), netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string, file string) (*Wallet, error) {

	var seed []byte
	for {
		// random seed
		seed = make([]byte, 64)
		_, err := Read(seed)
		if err != nil {
			return nil, err
		}

		// Ensure the seed can be used for generating a BLS keypair.
		_, err = generateConsensusKeys(seed)
		if err == nil {
			break
		}

		if err != io.EOF {
			return nil, err
		}
		// If not, we retry.
	}

	return LoadFromSeed(seed, netPrefix, db, fDecoys, fInputs, password, file)
}

func LoadFromSeed(seed []byte, netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string, file string) (*Wallet, error) {
	if len(seed) < 64 {
		return nil, errors.New("seed must be atleast 64 bytes in size")
	}
	err := saveSeed(seed, password, file)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateConsensusKeys(seed)
	if err != nil {
		return nil, err
	}

	w := &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		keyPair:       key.NewKeyPair(seed),
		consensusKeys: &consensusKeys,
		fetchDecoys:   fDecoys,
		fetchInputs:   fInputs,
	}

	// Check if this is a new wallet
	_, err = w.db.GetWalletHeight()
	if err == nil {
		return w, nil
	}

	if err != leveldb.ErrNotFound {
		return nil, err
	}

	// Add height of zero into database
	err = w.UpdateWalletHeight(0)
	if err != nil {
		return nil, err
	}

	return w, nil
}

func LoadFromFile(netPrefix byte, db *database.DB, fDecoys transactions.FetchDecoys, fInputs FetchInputs, password string, file string) (*Wallet, error) {

	seed, err := fetchSeed(password, file)
	if err != nil {
		return nil, err
	}

	consensusKeys, err := generateConsensusKeys(seed)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		db:            db,
		netPrefix:     netPrefix,
		keyPair:       key.NewKeyPair(seed),
		consensusKeys: &consensusKeys,
		fetchDecoys:   fDecoys,
		fetchInputs:   fInputs,
	}, nil
}

func (w *Wallet) CheckWireBlock(blk block.Block) (uint64, uint64, error) {
	// Ensure this block is at the height we expect it to be
	walletHeight, err := w.GetSavedHeight()
	if err != nil {
		return 0, 0, err
	}

	if blk.Header.Height != walletHeight {
		return 0, 0, fmt.Errorf("mismatch between block height and wallet height\nblock height: %v - wallet height: %v\n", blk.Header.Height, walletHeight)
	}

	spentCount, err := w.CheckWireBlockSpent(blk)
	if err != nil {
		return 0, 0, err
	}

	receivedCount, err := w.CheckWireBlockReceived(blk)
	if err != nil {
		return 0, 0, err
	}

	err = w.UpdateWalletHeight(blk.Header.Height + 1)
	if err != nil {
		return 0, 0, err
	}

	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, 0, err
	}

	if err := w.db.UpdateLockedInputs(privSpend.Bytes(), blk.Header.Height); err != nil {
		return 0, 0, err
	}

	return spentCount, receivedCount, nil
}

func (w *Wallet) CheckUnconfirmedBalance(txs []transactions.Transaction) (uint64, error) {
	privView, err := w.keyPair.PrivateView()
	if err != nil {
		return 0, err
	}

	var balance uint64
	for _, tx := range txs {
		for i, output := range tx.StandardTx().Outputs {
			if _, ok := w.keyPair.DidReceiveTx(tx.StandardTx().R, output.PubKey, uint32(i)); !ok {
				continue
			}

			var amount uint64
			if transactions.ShouldEncryptValues(tx) {
				amountScalar := transactions.DecryptAmount(output.EncryptedAmount, tx.StandardTx().R, uint32(i), *privView)
				amount = amountScalar.BigInt().Uint64()
			} else {
				amount = output.EncryptedAmount.BigInt().Uint64()
			}

			balance += amount
		}
	}

	return balance, nil
}

func (w *Wallet) Balance() (uint64, uint64, error) {
	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, 0, err
	}
	unlockedBalance, lockedBalance, err := w.db.FetchBalance(privSpend.Bytes())
	return unlockedBalance, lockedBalance, nil
}

// FetchTxHistory will return a slice containing information about all
// transactions made and received with this wallet.
func (w *Wallet) FetchTxHistory() ([]txrecords.TxRecord, error) {
	return w.db.FetchTxRecords()
}

func (w *Wallet) GetSavedHeight() (uint64, error) {
	return w.db.GetWalletHeight()
}

func (w *Wallet) UpdateWalletHeight(newHeight uint64) error {
	return w.db.UpdateWalletHeight(newHeight)
}

func (w *Wallet) PublicKey() key.PublicKey {
	return *w.keyPair.PublicKey()
}

func (w *Wallet) PublicAddress() (string, error) {
	pubAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	if err != nil {
		return "", err
	}
	return pubAddr.String(), nil
}

func (w *Wallet) ConsensusKeys() key.ConsensusKeys {
	return *w.consensusKeys
}

func (w *Wallet) PrivateSpend() ([]byte, error) {
	privateSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return nil, err
	}

	return privateSpend.Bytes(), nil
}

// ClearDatabase will remove all info from the database.
func (w *Wallet) ClearDatabase() error {
	return w.db.Clear()
}
