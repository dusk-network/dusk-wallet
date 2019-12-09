package transactions

import (
	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-wallet/key"
)

// NoteOutputTransparent is a note with a transparent value
type NoteOutputTransparent struct {
	// Value holds the transparent amount that can be spent
	Value float64

	// R is the randomness for the key generation.
	//
	// R := r . G
	R ristretto.Point

	// PubKey is a one-time pubkey that refers to the owner
	// of the note.
	//
	// PrivateSpend := scalarDerive(H(r))
	// PrivateView := scalarDerive(H(PrivateSpend))
	// PublicSpend := PrivateSpend . G
	// PublicView := PrivateView . G
	// PubKey := { PublicSpend, PublicView }
	PubKey key.StealthAddress

	// Idx refers to the position of the note in the tree.
	Idx uint64
}

// Transparent is true for transparent notes
func (no *NoteOutputTransparent) Transparent() bool {
	return true
}
