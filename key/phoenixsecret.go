package key

import ristretto "github.com/bwesterb/go-ristretto"

type PhoenixSecret struct {
    A ristretto.Scalar
    B ristretto.Scalar
}
