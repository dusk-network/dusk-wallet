package transactions

import ristretto "github.com/bwesterb/go-ristretto"

type Input struct {
	TxID       []byte
	Commitment ristretto.Point
	KeyImage   []byte
	RingSig    []byte
}

// Add new input function to prove we own each input
// Plus add the decoys, decoys will come from the node itself
