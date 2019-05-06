package mlsag

import ristretto "github.com/bwesterb/go-ristretto"

type PubKeys struct {
	// Set to true if the set of pubKeys are decoys
	decoy bool
	// Vector of pubKeys
	keys []ristretto.Point
}

func (p *PubKeys) AddPubKey(key ristretto.Point) {
	p.keys = append(p.keys, key)
}

func (p *PubKeys) Len() int {
	return len(p.keys)
}
