package transactions

import (
	ristretto "github.com/bwesterb/go-ristretto"
	"github.com/dusk-network/dusk-wallet/key"
)

// NoteOutput represents any note that can be used as utxo
type NoteOutput interface {
	IsTransparent() bool
}

// GenerateStealthAddress generates a random r and create a sk/pk pair from it
func GenerateStealthAddress() (*key.Key, ristretto.Scalar, ristretto.Point) {
	// Fetch a random scalar
	r := ristretto.Scalar{}
	r.Rand()

	// Create a deterministic key pair from the generated scalar
	keyPair := key.NewKeyPair(r.Bytes())

	// Apply the generator to the scalar
	var rG ristretto.Point
	rG.ScalarMultBase(&r)

	return keyPair, r, rG
}
