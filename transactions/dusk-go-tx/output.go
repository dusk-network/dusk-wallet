package transactions

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Output defines an output in a transaction.
type Output struct {
	// Index denotes the position of this output in the transaction
	Index uint32
	// Commitment is the pedersen commitment to the underlying amount
	// In a bidding transaction, it is the amount in cleartext
	// For this reason, the size is varied. Once bidding transactions use Commitments,
	// The size will be changed to a fixed 32 bytes
	Commitment []byte // 32 bytes
	// DestKey is the one-time public key of the address that
	// the funds should be sent to.
	DestKey []byte // 32 bytes
	//EncryptedAmount is the amount that is being sent to the recipient
	// It is encrypted using a shared secret.
	EncryptedAmount []byte //32 bytes
	//EncryptedAmount is the mask that is being used in the commitment to the amount
	// It is encrypted using a shared secret.
	EncryptedMask []byte //32 bytes
}

// NewOutput constructs a new Output from the passed parameters.
func NewOutput(index uint32, destKey []byte) (*Output, error) {

	if len(destKey) != 32 {
		return nil, errors.New("destination key is not 32 bytes")
	}

	return &Output{
		Index:   index,
		DestKey: destKey,
	}, nil
}

// Encode an Output struct and write to w.
func (o *Output) Encode(w io.Writer) error {
	if err := encoding.WriteUint32(w, binary.LittleEndian, o.Index); err != nil {
		return err
	}
	if err := encoding.Write256(w, o.Commitment); err != nil {
		return err
	}
	if err := encoding.Write256(w, o.DestKey); err != nil {
		return err
	}
	if err := encoding.Write256(w, o.EncryptedAmount); err != nil {
		return err
	}
	return encoding.Write256(w, o.EncryptedMask)
}

// Decode an Output object from r into an output struct.
func (o *Output) Decode(r io.Reader) error {
	if err := encoding.ReadUint32(r, binary.LittleEndian, &o.Index); err != nil {
		return err
	}
	if err := encoding.Read256(r, &o.Commitment); err != nil {
		return err
	}
	if err := encoding.Read256(r, &o.DestKey); err != nil {
		return err
	}
	if err := encoding.Read256(r, &o.EncryptedAmount); err != nil {
		return err
	}
	return encoding.Read256(r, &o.EncryptedMask)
}

// Equals returns true if two outputs are the same
func (o *Output) Equals(out *Output) bool {
	if o == nil || out == nil {
		return false
	}
	if !bytes.Equal(o.Commitment, out.Commitment) {
		return false
	}
	if !bytes.Equal(o.DestKey, out.DestKey) {
		return false
	}
	if !bytes.Equal(o.EncryptedAmount, out.EncryptedAmount) {
		return false
	}
	return bytes.Equal(o.EncryptedMask, out.EncryptedMask)
}
