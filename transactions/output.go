package transactions

import (
	"dusk-wallet/key"
	"encoding/binary"

	"dusk-wallet/rangeproof"

	ristretto "github.com/bwesterb/go-ristretto"
)

type Output struct {
	amount ristretto.Scalar
	mask   ristretto.Scalar

	DestKey         ristretto.Point
	EncryptedAmount ristretto.Scalar
	Index           uint32
	EncryptedMask   ristretto.Scalar
	Commitment      ristretto.Point
	RangeProof      rangeproof.Proof
}

func newOutput(r, amount ristretto.Scalar, index uint32, pubKey key.PublicKey) (*Output, error) {
	output := &Output{
		amount: amount,
	}

	stealthAddr := pubKey.StealthAddress(r, index)
	output.DestKey = stealthAddr.P

	proof, err := rangeproof.Prove([]ristretto.Scalar{amount}, false)
	if err != nil {
		return nil, err
	}
	output.RangeProof = proof

	output.Commitment = proof.V[0].Value
	output.mask = proof.V[0].BlindingFactor

	//XXX: When serialising the rangeproof, we miss out the commitment since it will
	// in the Output

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
