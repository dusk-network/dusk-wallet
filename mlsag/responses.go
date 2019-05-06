package mlsag

import ristretto "github.com/bwesterb/go-ristretto"

type Responses []ristretto.Scalar

func (r *Responses) AddResponse(res ristretto.Scalar) {
	*r = append(*r, res)
}
