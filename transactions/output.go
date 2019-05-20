package transactions

import (
	"dusk-wallet/key"
	"encoding/binary"
	"errors"
	"io"

	"dusk-wallet/rangeproof"

	ristretto "github.com/bwesterb/go-ristretto"
)

type Output struct {
	amount ristretto.Scalar
	mask   ristretto.Scalar

	DestKey         key.StealthAddress
	EncryptedAmount ristretto.Scalar
	Index           uint32
	EncryptedMask   ristretto.Scalar
	Commitment      ristretto.Point
	RangeProof      rangeproof.Proof
}

func (o *Output) Encode(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, o.DestKey.P.Bytes())
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, o.EncryptedAmount.Bytes())
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, o.EncryptedMask.Bytes())
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, o.Commitment.Bytes())
	if err != nil {
		return err
	}

	return o.RangeProof.Encode(w)
}

func (o *Output) Decode(r io.Reader) error {
	if o == nil {
		return errors.New("struct is nil")
	}

	err := readerToPoint(r, &o.DestKey.P)
	if err != nil {
		return err
	}
	err = readerToScalar(r, &o.EncryptedAmount)
	if err != nil {
		return err
	}
	err = readerToScalar(r, &o.EncryptedMask)
	if err != nil {
		return err
	}
	err = readerToPoint(r, &o.Commitment)
	if err != nil {
		return err
	}
	return o.RangeProof.Decode(r)
}

func (o *Output) Equals(other Output) bool {
	ok := o.DestKey.P.Equals(&other.DestKey.P)
	if !ok {
		return ok
	}
	ok = o.EncryptedAmount.Equals(&other.EncryptedAmount)
	if !ok {
		return ok
	}
	ok = o.EncryptedMask.Equals(&other.EncryptedMask)
	if !ok {
		return ok
	}
	ok = o.Commitment.Equals(&other.Commitment)
	if !ok {
		return ok
	}
	return o.RangeProof.Equals(other.RangeProof)
}

func NewOutput(r, amount ristretto.Scalar, index uint32, pubKey key.PublicKey) (*Output, error) {
	output := &Output{
		amount: amount,
	}

	output.Index = index

	stealthAddr := pubKey.StealthAddress(r, index)
	output.DestKey = *stealthAddr

	proof, err := rangeproof.Prove([]ristretto.Scalar{amount}, false)
	if err != nil {
		return nil, err
	}
	output.RangeProof = proof

	output.Commitment = proof.V[0].Value
	output.mask = proof.V[0].BlindingFactor

	//XXX: When serialising the rangeproof, we miss out the commitment since it will
	// be in the Output

	output.EncryptedAmount = encryptAmount(output.amount, r, index, *pubKey.PubView)
	output.EncryptedMask = encryptMask(output.mask, r, index, *pubKey.PubView)

	return output, nil
}

// encAmount = amount + H(H(H(r*PubViewKey || index)))
func encryptAmount(amount, r ristretto.Scalar, index uint32, pubViewKey key.PublicView) ristretto.Scalar {
	rView := pubViewKey.ScalarMult(r)

	rViewIndex := append(rView.Bytes(), uint32ToBytes(index)...)

	var encryptKey ristretto.Scalar
	encryptKey.Derive(rViewIndex)
	encryptKey.Derive(encryptKey.Bytes())
	encryptKey.Derive(encryptKey.Bytes())

	var encryptedAmount ristretto.Scalar
	encryptedAmount.Add(&amount, &encryptKey)

	return encryptedAmount
}

// encMask = mask + H(H(r*PubViewKey || index))
func encryptMask(mask, r ristretto.Scalar, index uint32, pubViewKey key.PublicView) ristretto.Scalar {
	rView := pubViewKey.ScalarMult(r)
	rViewIndex := append(rView.Bytes(), uint32ToBytes(index)...)

	var encryptKey ristretto.Scalar
	encryptKey.Derive(rViewIndex)
	encryptKey.Derive(encryptKey.Bytes())

	var encryptedMask ristretto.Scalar
	encryptedMask.Add(&mask, &encryptKey)

	return encryptedMask
}

func uint32ToBytes(x uint32) []byte {
	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, x)
	return a
}
