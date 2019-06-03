package wallet

import (
	"crypto/rand"
	"dusk-wallet/key"
	"dusk-wallet/transactions/v1"
	"math/big"

	"github.com/bwesterb/go-ristretto"
)

// Number of mixins per ring. ringsize = mixin + 1
const numMixins = 7

// FetchInputs returns a slice of inputs such that Sum(Inputs)- Sum(Outputs) >= 0
// If > 0, then a change address is created for the remaining amount
type FetchInputs func(netPrefix byte, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error)

type Wallet struct {
	netPrefix   byte
	keyPair     *key.Key
	fetchDecoys transactions.FetchDecoys
	fetchInputs FetchInputs
}

func New(netPrefix byte, fDecoys transactions.FetchDecoys, fInputs FetchInputs) (*Wallet, error) {

	// random seed
	seed := make([]byte, 64)
	_, err := rand.Read(seed)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		netPrefix:   netPrefix,
		keyPair:     key.NewKeyPair(seed),
		fetchDecoys: fDecoys,
		fetchInputs: fInputs,
	}, nil
}

func (w *Wallet) NewStealthTx(fee int64) (*transactions.StealthTx, error) {
	tx, err := transactions.NewStealth(w.netPrefix, fee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// AddInputs adds up the total outputs and fee then fetches inputs to consolidate this
func (w *Wallet) AddInputs(tx *transactions.StealthTx) error {

	totalAmount := tx.Fee.BigInt().Int64() + tx.TotalSent.BigInt().Int64()

	inputs, changeAmount, err := w.fetchInputs(w.netPrefix, totalAmount, w.keyPair)
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

func (w *Wallet) Sign(tx *transactions.StealthTx) error {

	// Assuming user has added all of the outputs

	// Fetch Inputs
	err := w.AddInputs(tx)
	if err != nil {
		return err
	}

	// fetch decoys
	err = tx.AddDecoys(numMixins, w.fetchDecoys)
	if err != nil {
		return err
	}

	return tx.CalcCommToZero()
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
