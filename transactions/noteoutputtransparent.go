package transactions

import (
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-wallet/key"
)

// NoteOutputTransparent is a note with a transparent value
type NoteOutputTransparent struct {
	// Value holds the transparent amount that can be spent
	Value big.Int

	// R is the randomness for the key generation.
	//
	// R := r . G
	R ristretto.Point

	// PubKey is a one-time pubkey generated key.
	//
	// PrivateSpend := scalarDerive(H(r))
	// PrivateView := scalarDerive(H(PrivateSpend))
	// PublicSpend := PrivateSpend . G
	// PublicView := PrivateView . G
	// PubKey := { PublicSpend, PublicView }
	PubKey key.PublicKey

	// Idx refers to the position of the note in the tree.
	Idx uint64
}

// NewNoteOutputTransparent will create a new utxo phoenix note
func NewNoteOutputTransparent(value *big.Int) *NoteOutputTransparent {
	keyPair, _, rG := GenerateStealthAddress()

	return &NoteOutputTransparent{
		Value:  *value,
		R:      rG,
		PubKey: *keyPair.PublicKey(),
	}
}

// IsTransparent is true for transparent notes
func (no *NoteOutputTransparent) IsTransparent() bool {
	return true
}
