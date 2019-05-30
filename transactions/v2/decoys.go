package transactions

import (
	"dusk-wallet/mlsag"
	"errors"

	"github.com/bwesterb/go-ristretto"
)

// Decoy represents the information needed
// to add a decoy user to the ring of mlsag members
type Decoy struct {
	// Commitment is the commitment to the amount
	// in the output of the transaction you are adding as a decoy
	Commitment ristretto.Point
	// PubKey is the destination key in the output
	PubKey ristretto.Point
	// Offset is the global counter of the output
	Offset []byte
}

// FetchDecoys returns a slice of decoy
// Note: calling this function repeatedly should return a different slice of decoys
// The sampling technique in this method is important for anonymity
type FetchDecoys func(numMixins int) Decoys

type Decoys []Decoy

// ToMLSAG converts each decoy into a mlsag pubkey and an offset.
func (d Decoys) ToMLSAG() ([]mlsag.PubKeys, [][]byte, error) {
	if len(d) < 1 {
		return nil, nil, errors.New("there are no decoys available to convert to MLSAG")
	}

	mlsagKeys := make([]mlsag.PubKeys, len(d))
	globalOffsets := make([][]byte, len(d))

	for i := range d {
		decoy := d[i]

		// Add decoy Destination key
		mlsagKeys[i].AddPubKey(decoy.PubKey)

		// Add Commitment as intermediate commitment to zero
		mlsagKeys[i].AddPubKey(decoy.Commitment)

		// Add offset
		globalOffsets[i] = decoy.Offset
	}

	return mlsagKeys, globalOffsets, nil
}
