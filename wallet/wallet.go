package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/dusk-network/dusk-wallet/block"
	"github.com/dusk-network/dusk-wallet/database"
	"github.com/dusk-network/dusk-wallet/key"
	"github.com/dusk-network/dusk-wallet/transactions"
	zkproof "github.com/dusk-network/dusk-zkproof"
	"golang.org/x/crypto/sha3"

	"github.com/bwesterb/go-ristretto"
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

	// random seed
	seed := make([]byte, 64)
	_, err := Read(seed)
	if err != nil {
		return nil, err
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

func (w *Wallet) NewStandardTx(fee int64) (*transactions.Standard, error) {
	tx, err := transactions.NewStandard(0, w.netPrefix, fee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (w *Wallet) NewStakeTx(fee int64, lockTime uint64, amount ristretto.Scalar) (*transactions.Stake, error) {
	edPubBytes := w.consensusKeys.EdPubKeyBytes
	blsPubBytes := w.consensusKeys.BLSPubKeyBytes
	tx, err := transactions.NewStake(0, w.netPrefix, fee, lockTime, edPubBytes, blsPubBytes)
	if err != nil {
		return nil, err
	}

	// Send locked stake amount to self
	walletAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	err = tx.AddOutput(*walletAddr, amount)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (w *Wallet) NewBidTx(fee int64, lockTime uint64, amount ristretto.Scalar) (*transactions.Bid, error) {
	privateSpend, err := w.keyPair.PrivateSpend()
	privateSpend.Bytes()

	// TODO: index is currently set to be zero.
	// To avoid any privacy implications, the wallet should increment
	// the index by how many bidding txs are seen
	mBytes := generateM(privateSpend.Bytes(), 0)
	tx, err := transactions.NewBid(0, w.netPrefix, fee, lockTime, mBytes)
	if err != nil {
		return nil, err
	}

	// Send bid amount to self
	walletAddr, err := w.keyPair.PublicKey().PublicAddress(w.netPrefix)
	err = tx.AddOutput(*walletAddr, amount)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (w *Wallet) CheckWireBlock(blk block.Block) (uint64, uint64, error) {
	// Ensure this block is at the height we expect it to be
	walletHeight, err := w.GetSavedHeight()
	if err != nil {
		return 0, 0, err
	}

	if blk.Header.Height != walletHeight {
		return 0, 0, errors.New("last seen block does not precede provided block")
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
			if shouldEncryptValues(tx) {
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

// AddInputs adds up the total outputs and fee then fetches inputs to consolidate this
func (w *Wallet) AddInputs(tx *transactions.Standard) error {
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

func (w *Wallet) Sign(tx SignableTx) error {
	// Assuming user has added all of the outputs
	standardTx := tx.StandardTx()

	// Fetch Inputs
	err := w.AddInputs(standardTx)
	if err != nil {
		return err
	}

	// Fetch decoys
	err = standardTx.AddDecoys(numMixins, w.fetchDecoys)
	if err != nil {
		return err
	}

	if err := tx.Prove(); err != nil {
		return err
	}

	// Remove inputs from the db, to prevent accidental double-spend attempts
	// when sending transactions quickly after one another.
	for _, input := range tx.StandardTx().Inputs {
		pubKey, err := w.db.Get(input.KeyImage.Bytes())
		if err == leveldb.ErrNotFound {
			continue
		}
		if err != nil {
			return err
		}

		w.db.RemoveInput(pubKey, input.KeyImage.Bytes())
	}

	return nil
}

func (w *Wallet) Balance() (uint64, uint64, error) {
	privSpend, err := w.keyPair.PrivateSpend()
	if err != nil {
		return 0, 0, err
	}
	unlockedBalance, lockedBalance, err := w.db.FetchBalance(privSpend.Bytes())
	return unlockedBalance, lockedBalance, nil
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

// Save saves the seed to a dat file
func saveSeed(seed []byte, password string, file string) error {
	// Overwriting a seed file may cause loss of funds
	if _, err := os.Stat(file); err == nil {
		return ErrSeedFileExists
	}

	digest := sha3.Sum256([]byte(password))

	c, err := aes.NewCipher(digest[:])
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	return ioutil.WriteFile(file, gcm.Seal(nonce, nonce, seed, nil), 0777)
}

//Modified from https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
func fetchSeed(password string, file string) ([]byte, error) {

	digest := sha3.Sum256([]byte(password))

	ciphertext, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(digest[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, err
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	seed, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return seed, nil
}

func generateConsensusKeys(seed []byte) (key.ConsensusKeys, error) {
	// Consensus keys require >80 bytes of seed, so we will hash seed twice and concatenate
	// both hashes to get 128 bytes

	seedHash := sha3.Sum512(seed)
	secondSeedHash := sha3.Sum512(seedHash[:])

	consensusSeed := append(seedHash[:], secondSeedHash[:]...)

	return key.NewConsensusKeysFromBytes(consensusSeed)
}

func generateM(PrivateSpend []byte, index uint32) []byte {

	// To make K deterministic
	// We will calculate K = PrivateSpend || Index
	// Index is the number of Bidding transactions that has
	// been initiated. This information should be available to the wallet
	// M = H(K)

	numBidTxsSeen := make([]byte, 4)
	binary.BigEndian.PutUint32(numBidTxsSeen, index)

	KBytes := append(PrivateSpend, numBidTxsSeen...)

	// Encode K as a ristretto Scalar
	var k ristretto.Scalar
	k.Derive(KBytes)

	m := zkproof.CalculateM(k)
	return m.Bytes()
}

func (w *Wallet) ReconstructK() (ristretto.Scalar, error) {
	zeroPadding := make([]byte, 4)
	privSpend, err := w.PrivateSpend()
	if err != nil {
		return ristretto.Scalar{}, err
	}

	kBytes := append(privSpend, zeroPadding...)
	var k ristretto.Scalar
	k.Derive(kBytes)
	return k, nil
}
