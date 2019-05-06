package transactions

import (
	"dusk-wallet/mlsag"

	ristretto "github.com/bwesterb/go-ristretto"
)

type Input struct {
	TxID         []byte
	Commitment   ristretto.Point
	amount, mask ristretto.Scalar

	// Onetime pubkey
	Pubkey  ristretto.Point
	privKey ristretto.Scalar

	PseudoCommitment ristretto.Point
	pseudoMask       ristretto.Scalar

	Proof *mlsag.Proof
	Sig   *mlsag.Signature
}

func NewInput(txid []byte, commitment ristretto.Point, amount, mask ristretto.Scalar, pubkey ristretto.Point, privKey ristretto.Scalar) *Input {
	return &Input{
		TxID:       txid,
		Commitment: commitment,
		amount:     amount,
		mask:       mask,
		Proof:      &mlsag.Proof{},
	}
}

func (i *Input) AddDecoyKeyVector(pubKeys mlsag.PubKeys) {
	i.Proof.AddDecoy(pubKeys)
}

func (i *Input) AddSecretKeyVector(p mlsag.PrivKeys) {
	i.Proof.AddSecret(p)
}

func (i *Input) Prove() error {
	sig, err := i.Proof.Prove()
	if err != nil {
		return err
	}
	i.Sig = sig
	return nil
}

func (i *Input) Verify() (bool, error) {
	return i.Sig.Verify()
}
