package transactions

import (
	"dusk-wallet/key"

	ristretto "github.com/bwesterb/go-ristretto"
)

type StealthTx struct {
	r ristretto.Scalar
	R ristretto.Point

	Inputs  []*Input
	Outputs []*Output
	Fee     ristretto.Scalar

	index     uint32
	netPrefix byte

	// Proof for commitment to zero
	signature []byte
}

func newTransaction(netPrefix byte) *StealthTx {

	tx := &StealthTx{}

	tx.index = 1

	tx.netPrefix = netPrefix

	// randomly generated r
	var r ristretto.Scalar
	r.Rand()
	tx.r = r

	var R ristretto.Point
	R.ScalarMultBase(&r)
	tx.R = R

	return tx
}

// XXX:For now just add the commitment, so we can prove commitment to zero
func (s *StealthTx) AddInput(commitment ristretto.Point) error {

	s.Inputs = append(s.Inputs, &Input{
		Commitment: commitment,
	})

	return nil
}

func calcCommToZero(inputs []*Input, outputs []*Output) ristretto.Point {
	var sumInputComm ristretto.Point
	for i := range inputs {
		inComm := inputs[i].Commitment
		sumInputComm.Add(&inComm, &sumInputComm)
	}
	var sumOutputComm ristretto.Point
	for i := range outputs {
		outComm := outputs[i].Commitment
		sumOutputComm.Add(&outComm, &sumOutputComm)
	}

	var commToZero ristretto.Point
	commToZero.Sub(&sumOutputComm, &sumInputComm)

	return commToZero
}

func (s *StealthTx) AddOutput(pubAddr key.PublicAddress, amount ristretto.Scalar) error {
	pubKey, err := pubAddr.ToKey(s.netPrefix)
	if err != nil {
		return err
	}

	output := newOutput(s.r, amount, s.index, *pubKey)
	s.Outputs = append(s.Outputs, output)

	s.index = s.index + 1

	return nil
}
