package block

import (
	"bytes"
)

// Certificate defines a block certificate made as a result from the consensus.
type Certificate struct {
	Round             uint64
	Step              uint8 // Step the agreement terminated at (1 byte)
	BlockHash         []byte
	StepOneBatchedSig []byte // Batched BLS signature of the block reduction phase (33 bytes)
	StepTwoBatchedSig []byte
	StepOneCommittee  uint64 // Binary representation of the committee members who voted in favor of this block (8 bytes)
	StepTwoCommittee  uint64
}

func EmptyCertificate() *Certificate {
	return &Certificate{
		Round:             0,
		Step:              0,
		BlockHash:         make([]byte, 32),
		StepOneBatchedSig: make([]byte, 33),
		StepTwoBatchedSig: make([]byte, 33),
		StepOneCommittee:  0,
		StepTwoCommittee:  0,
	}
}

// Equals returns true if both certificates are equal
func (c *Certificate) Equals(other *Certificate) bool {
	if other == nil {
		return false
	}

	if c.Round != other.Round {
		return false
	}

	if c.Step != other.Step {
		return false
	}

	if !bytes.Equal(c.BlockHash, other.BlockHash) {
		return false
	}

	if !bytes.Equal(c.StepOneBatchedSig, other.StepOneBatchedSig) {
		return false
	}

	if !bytes.Equal(c.StepTwoBatchedSig, other.StepTwoBatchedSig) {
		return false
	}

	if c.StepOneCommittee != other.StepOneCommittee {
		return false
	}

	if c.StepTwoCommittee != other.StepTwoCommittee {
		return false
	}

	return true
}
