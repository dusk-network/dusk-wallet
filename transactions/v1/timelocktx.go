package transactions

import (
	"encoding/binary"
	"io"
)

type TimeLock struct {
	*StealthTx
	Lock int64
}

func NewTimeLock(netPrefix byte, fee, lock int64) (*TimeLock, error) {
	stx, err := NewStealth(netPrefix, fee)
	if err != nil {
		return nil, err
	}

	return &TimeLock{
		stx,
		lock,
	}, nil
}

func (tl *TimeLock) Encode(w io.Writer) error {
	err := tl.StealthTx.Encode(w)
	if err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, tl.Lock)
}
func (tl *TimeLock) Decode(r io.Reader) error {
	err := tl.StealthTx.Decode(r)
	if err != nil {
		return err
	}
	return binary.Read(r, binary.BigEndian, &tl.Lock)
}
