package main

import (
	"dusk-wallet/database"
	"dusk-wallet/key"
	"dusk-wallet/mlsag"
	"dusk-wallet/transactions/v3"
	"dusk-wallet/wallet/v3"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/bwesterb/go-ristretto"
)

// Testing functionality

func main() {

	mainnet := byte(1)

	// Create/Load Database
	db, err := database.New(strconv.Itoa(int(mainnet)))
	if err != nil {
		panic(fmt.Sprintf("unexpected error opening wallet %s", err.Error()))
	}

	// Load wallet
	w, err := wallet.New(mainnet, db, fetchDecoys, fetchInputs)
	if err != nil {
		panic(fmt.Sprintf("unexpected error opening wallet %s", err.Error()))
	}
	senderPubKey := w.PublicKey()
	senderPubAddr, err := senderPubKey.PublicAddress(mainnet)
	if err != nil {
		panic(fmt.Sprintf("could not determine the public address %s", err.Error()))
	}

	// Add Fake inputs to wallet
	_, err = LoadFakeInputs(db, mainnet, w)
	if err != nil {
		panic(fmt.Sprintf("could not add fake inputs %s", err.Error()))
	}

	// Create tx
	// Attach a fee of 0 DUSK
	tx, err := w.NewStandardTx(0)
	if err != nil {
		panic(fmt.Sprintf("unexpected error creating tx %s", err.Error()))
	}

	// We send Alice 50 DUSK
	alice := newUser("alice")
	tx.AddOutput(alice.sendAddr(mainnet), intToScalar(50))

	// We send bob 23 DUSK
	bob := newUser("bob")
	tx.AddOutput(bob.sendAddr(mainnet), intToScalar(23))

	// // We send ourselves 20 DUSK
	tx.AddOutput(*senderPubAddr, intToScalar(29))

	err = w.Sign(tx)
	if err != nil {
		panic(fmt.Sprintf("could not sign tx %s", err.Error()))
	}

	// Tx hash
	txid, err := tx.Hash()
	if err != nil {
		panic(fmt.Errorf("cannot get hash of txid %s", err.Error()))
	}
	fmt.Println("txid hash", hex.EncodeToString(txid))

	// Add tx to a block
	var blk wallet.Block
	blk.AddStandardTx(*tx)

	_, err = w.CheckBlockSpent(blk)
	if err != nil {
		panic(fmt.Errorf("could not check block spent %s", err.Error()))
	}
	_, err = w.CheckBlockReceived(blk)
	if err != nil {
		panic(fmt.Errorf("could not check block received %s", err.Error()))
	}
}

/*

We define two functions:

- FetchDecoys will fetch possible decoy values from the node

- FetchInputs will fetch inputs for the wallet to use. Note that this function assumes that all inputs stored belong to
the same wallet. We can have it take a parameter to separate wallets. Maybe one database per wallet.
Where the parameter is the db name
*/

func fetchDecoys(numMixins int) []mlsag.PubKeys {
	var pubKeys []mlsag.PubKeys
	for i := 0; i < numMixins; i++ {
		pubKeyVector := generateDualKey()
		pubKeys = append(pubKeys, pubKeyVector)
	}
	return pubKeys
}

func generateDualKey() mlsag.PubKeys {
	pubkeys := mlsag.PubKeys{}

	var primaryKey ristretto.Point
	primaryKey.Rand()
	pubkeys.AddPubKey(primaryKey)

	var secondaryKey ristretto.Point
	secondaryKey.Rand()
	pubkeys.AddPubKey(secondaryKey)

	return pubkeys
}

func fetchInputs(netPrefix byte, db *database.DB, totalAmount int64, key *key.Key) ([]*transactions.Input, int64, error) {
	// Fetch all inputs from database that are >= totalAmount
	// returns error if inputs do not add up to total amount
	privSpend, err := key.PrivateSpend()
	if err != nil {
		return nil, 0, err
	}
	return db.FetchInputs(privSpend.Bytes(), totalAmount)
}

// Convenience functions

type user struct {
	*key.Key
}

func newUser(name string) *user {
	return &user{
		key.NewKeyPair([]byte(name)),
	}
}
func (u *user) sendAddr(netPrefix byte) key.PublicAddress {
	pubAddr, err := u.PublicKey().PublicAddress(netPrefix)
	if err != nil {
		panic(err)
	}
	return *pubAddr
}

func intToScalar(amount int64) ristretto.Scalar {
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(amount))
	return x
}

func LoadFakeInputs(db *database.DB, netPrefix byte, w *wallet.Wallet) (uint64, error) {

	fmt.Println("Loading fake inputs")

	// Creates a block and sends transactions to self
	// Then runs wallet's block checker, which loads all outputs in the database
	senderKey := w.PublicKey()
	senderAddr, err := senderKey.PublicAddress(netPrefix)
	if err != nil {
		return 0, err
	}
	tx := w.NewCoinbaseTx()
	err = addOutputsToTx(*senderAddr, 120, tx)
	if err != nil {
		return 0, err
	}
	var blk wallet.Block
	blk.AddCoinbaseTx(*tx)

	return w.CheckBlockReceived(blk)
}

func addOutputsToTx(receiver key.PublicAddress, amount int64, tx *transactions.CoinbaseTx) error {
	var x ristretto.Scalar
	x.SetBigInt(big.NewInt(amount))
	for i := 0; i < 10; i++ {
		err := tx.AddReward(receiver, x)
		if err != nil {
			return err
		}
	}
	return nil
}

func R(r ristretto.Scalar) ristretto.Point {
	var R ristretto.Point
	R.ScalarMultBase(&r)
	return R
}
