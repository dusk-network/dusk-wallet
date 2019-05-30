package transactions

import (
	"dusk-wallet/key"
	dtx "dusk-wallet/transactions/dusk-go-tx"
	"encoding/binary"

	"github.com/bwesterb/go-ristretto"
)

type Output struct {
	baseOutput dtx.Output

	Commitment ristretto.Point
	amount     ristretto.Scalar
	mask       ristretto.Scalar

	// PubKey refers to the destination key of the receiver
	PubKey key.StealthAddress

	// Index denotes the position that this output is in the
	// transaction. This is different to the Offset which denotes the
	// position that this output is in, from the start from the blockchain
	Index uint32

	EncryptedAmount ristretto.Scalar
	EncryptedMask   ristretto.Scalar
}

func NewOutput(r, amount ristretto.Scalar, index uint32, pubKey key.PublicKey) *Output {
	output := &Output{
		amount: amount,
	}

	output.setIndex(index)

	stealthAddr := pubKey.StealthAddress(r, index)
	output.setDestKey(stealthAddr)

	return output
}

func (o *Output) setIndex(index uint32) {
	o.Index = index
	o.baseOutput.Index = index
}
func (o *Output) setDestKey(stealthAddr *key.StealthAddress) {
	o.PubKey = *stealthAddr
	o.baseOutput.DestKey = stealthAddr.P.Bytes()
}
func (o *Output) setCommitment(comm ristretto.Point) {
	o.Commitment = comm
	o.baseOutput.Commitment = comm.Bytes()
}
func (o *Output) setMask(mask ristretto.Scalar) {
	o.mask = mask
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
