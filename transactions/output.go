package transactions

import (
	"dusk-wallet/key"
	"encoding/binary"

	ristretto "github.com/bwesterb/go-ristretto"
	"gitlab.dusk.network/dusk-core/dusk-go/pkg/crypto/rangeproof"
)

type Output struct {
	DestKey         ristretto.Point
	EncryptedAmount ristretto.Scalar
	EncryptedMask   ristretto.Scalar
	Commitment      ristretto.Point
	RangeProof      rangeproof.Proof
}

func newOutput(r, amount ristretto.Scalar, index uint32, pubKey key.PublicKey) *Output {

	output := &Output{}

	stealthAddr := pubKey.StealthAddress(r, index)
	output.DestKey = stealthAddr.P

	mask, commitment := commit(amount)
	output.Commitment = commitment

	output.EncryptedAmount = encryptAmount(amount, r, index, *pubKey.PubView)

	output.EncryptedMask = encryptMask(mask, r, index, *pubKey.PubView)

	// TODO: Add rangeproof for output
	// XXX: We need the rangeproof commitment to match with the commitment in the output.
	// Modify rangeproof to return commitment, mask and proof
	// Then do rangeproof first
	// rangeproof.Prove()

	return output
}

// Commitment = amount * G + mask * H
func commit(amount ristretto.Scalar) (ristretto.Scalar, ristretto.Point) {
	var commitment ristretto.Point
	commitment.ScalarMultBase(&amount)

	var H ristretto.Point
	H.Derive([]byte("blind"))

	var mask ristretto.Scalar
	mask.Rand()

	H.ScalarMult(&H, &mask)

	// commitment = amount * G + mask * H
	commitment.Add(&commitment, &H)

	return mask, commitment
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
