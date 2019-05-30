package transactions

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"gitlab.dusk.network/dusk-core/dusk-go/pkg/p2p/wire/encoding"
)

// Input defines an input in a standard transaction.
type Input struct {
	// KeyImage is the image of the key that is being used to
	// sign the transaction
	KeyImage []byte // 32 bytes
	// Offsets denotes a global counter, referring to the position that an input
	// was as an output. In reference to every other input, from block zero.
	// For example; 5 would indicate that this is the fifth input to be created in the blockchain
	// Indices holds all of the inputs being used; real and decoy
	Offsets [][]byte // var bytes
	// PseudoCommitment is a intermediate commitment, used to create the balance proof
	PseudoCommitment []byte // 32 bytes
	// Signature refers to mlsag dual-key signature. The public keys will be taken from a database
	// indexed by the offsets. Therefore the public keys will not be serialised in the signature
	Signature []byte // var bytes
}

// NewInput constructs a new Input from the passed parameters.
func NewInput(keyImage []byte) (*Input, error) {

	if len(keyImage) != 32 {
		return nil, errors.New("key image does not equal 32 bytes")
	}

	return &Input{
		KeyImage: keyImage,
	}, nil
}

func (i *Input) AddInput(globalOffset []byte) {
	i.Offsets = append(i.Offsets, globalOffset)
}

// Encode an Input object into an io.Writer.
func (i *Input) Encode(w io.Writer) error {
	if err := encoding.Write256(w, i.KeyImage); err != nil {
		return err
	}

	lenI := uint64(len(i.Offsets))
	if err := encoding.WriteUint64(w, binary.LittleEndian, lenI); err != nil {
		return err
	}

	for k := uint64(0); k < lenI; k++ {
		if err := encoding.WriteVarBytes(w, i.Offsets[k]); err != nil {
			return err
		}
	}

	if err := encoding.Write256(w, i.PseudoCommitment); err != nil {
		return err
	}
	return encoding.WriteVarBytes(w, i.Signature)
}

// Decode an Input object from a io.reader.
func (i *Input) Decode(r io.Reader) error {
	if err := encoding.Read256(r, &i.KeyImage); err != nil {
		return err
	}

	var lenI uint64
	if err := encoding.ReadUint64(r, binary.LittleEndian, &lenI); err != nil {
		return err
	}

	i.Offsets = make([][]byte, lenI)
	for k := uint64(0); k < lenI; k++ {
		if err := encoding.ReadVarBytes(r, &i.Offsets[k]); err != nil {
			return err
		}
	}

	if err := encoding.Read256(r, &i.PseudoCommitment); err != nil {
		return err
	}
	return encoding.ReadVarBytes(r, &i.Signature)
}

// Equals returns true if two inputs are the same
func (i *Input) Equals(in *Input) bool {
	if in == nil || i == nil {
		return false
	}

	if !bytes.Equal(i.KeyImage, in.KeyImage) {
		return false
	}

	if len(i.Offsets) != len(in.Offsets) {
		return false
	}

	for k := range i.Offsets {
		if !bytes.Equal(i.Offsets[k], in.Offsets[k]) {
			return false
		}
	}

	if !bytes.Equal(i.PseudoCommitment, in.PseudoCommitment) {
		return false
	}
	return bytes.Equal(i.Signature, in.Signature)
}
