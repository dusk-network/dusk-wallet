package transactions

import (
	"bytes"
	"dusk-wallet/mlsag"
	"encoding/binary"
	"errors"
	"io"

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

	Proof    *mlsag.DualKey
	keyImage ristretto.Point
	Sig      *mlsag.Signature
}

func NewInput(txid []byte, commitment ristretto.Point, amount, mask ristretto.Scalar, pubkey ristretto.Point, privKey ristretto.Scalar) *Input {
	return &Input{
		TxID:       txid,
		Commitment: commitment,
		amount:     amount,
		mask:       mask,
		Proof:      mlsag.NewDualKey(),
	}
}

func (i *Input) SetPrimaryKey(key ristretto.Scalar) {
	i.Proof.SetPrimaryKey(key)
}
func (i *Input) SetCommToZer(key ristretto.Scalar) {
	i.Proof.SetCommToZero(key)
}

func (i *Input) Prove() error {
	sig, keyImage, err := i.Proof.Prove()
	if err != nil {
		return err
	}
	i.keyImage = keyImage
	i.Sig = sig
	return nil
}

func (i *Input) Verify() (bool, error) {
	return i.Sig.Verify([]ristretto.Point{i.keyImage})
}

func (i *Input) Encode(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, i.TxID)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, i.Commitment.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, i.Pubkey.Bytes())
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, i.PseudoCommitment.Bytes())
	if err != nil {
		return err
	}
	return i.Sig.Encode(w, false)
}

func (i *Input) Decode(r io.Reader) error {

	if i == nil {
		return errors.New("struct is nil")
	}

	var x [32]byte
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	i.TxID = make([]byte, 32)
	copy(i.TxID[:], x[:])

	err = readerToPoint(r, &i.Commitment)
	if err != nil {
		return err
	}

	err = readerToPoint(r, &i.Pubkey)
	if err != nil {
		return err
	}

	err = readerToPoint(r, &i.PseudoCommitment)
	if err != nil {
		return err
	}
	i.Sig = &mlsag.Signature{}
	return i.Sig.Decode(r, false)
}

func (i *Input) Equals(other Input) bool {
	ok := bytes.Equal(i.TxID, other.TxID)
	if !ok {
		return ok
	}
	ok = i.Commitment.Equals(&other.Commitment)
	if !ok {
		return ok
	}
	ok = i.Pubkey.Equals(&other.Pubkey)
	if !ok {
		return ok
	}
	ok = i.PseudoCommitment.Equals(&other.PseudoCommitment)
	if !ok {
		return ok
	}
	return i.Sig.Equals(*other.Sig, false)
}

func readerToPoint(r io.Reader, p *ristretto.Point) error {
	var x [32]byte
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	ok := p.SetBytes(&x)
	if !ok {
		return errors.New("point not encodable")
	}
	return nil
}
func readerToScalar(r io.Reader, s *ristretto.Scalar) error {
	var x [32]byte
	err := binary.Read(r, binary.BigEndian, &x)
	if err != nil {
		return err
	}
	s.SetBytes(&x)
	return nil
}
