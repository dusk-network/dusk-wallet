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
	// Index denotes a global counter, referring to the position that an input
	// was as an output. In reference to every other input, from block zero.
	// For example; 5 would indicate that this is the fifth input to be created in the blockchain
	// Indices holds all of the inputs being used; real and decoy
	Indices [][]byte // var byte
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

func (i *Input) AddInput(index []byte) {
	i.Indices = append(i.Indices, index)
}

// Encode an Input object into an io.Writer.
func (i *Input) Encode(w io.Writer) error {
	if err := encoding.Write256(w, i.KeyImage); err != nil {
		return err
	}

	lenI := uint64(len(i.Indices))
	if err := encoding.WriteUint64(w, binary.LittleEndian, lenI); err != nil {
		return err
	}

	for k := uint64(0); k < lenI; k++ {
		if err := encoding.WriteVarBytes(w, i.Indices[k]); err != nil {
			return err
		}
	}

	return nil
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

	i.Indices = make([][]byte, lenI)
	for k := uint64(0); k < lenI; k++ {
		if err := encoding.ReadVarBytes(r, &i.Indices[k]); err != nil {
			return err
		}
	}

	return nil
}

// Equals returns true if two inputs are the same
func (i *Input) Equals(in *Input) bool {
	if in == nil || i == nil {
		return false
	}

	if !bytes.Equal(i.KeyImage, in.KeyImage) {
		return false
	}

	if len(i.Indices) != len(in.Indices) {
		return false
	}

	for k := range i.Indices {
		if !bytes.Equal(i.Indices[k], in.Indices[k]) {
			return false
		}
	}

	// Omit Index equality; same input could be at two different
	// places in a tx

	return true
}
