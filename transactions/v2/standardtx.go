package transactions

import (
	"dusk-wallet/key"
	"dusk-wallet/rangeproof"
	dtx "dusk-wallet/transactions/dusk-go-tx"
	"errors"
	"math/big"

	"github.com/bwesterb/go-ristretto"
)

const maxInputs = 2000
const maxOutputs = 32

type StandardTx struct {
	f FetchDecoys

	baseTx dtx.Standard

	r ristretto.Scalar
	R ristretto.Point

	Inputs  []*Input
	Outputs []*Output
	Fee     ristretto.Scalar

	index     uint32
	netPrefix byte

	RangeProof rangeproof.Proof

	TotalSent ristretto.Scalar
}

func NewStandard(netPrefix byte, fee int64) (*StandardTx, error) {

	tx := &StandardTx{}

	tx.TotalSent.SetZero()

	// Index for subaddresses
	tx.index = 0

	// prefix to signify testnet/mainnet
	tx.netPrefix = netPrefix

	// randomly generated nonce - r
	var r ristretto.Scalar
	r.Rand()
	tx.setTxPubKey(r)

	// Set fee
	err := tx.setTxFee(fee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *StandardTx) setTxPubKey(r ristretto.Scalar) {
	s.r = r
	s.R.ScalarMultBase(&r)
	s.baseTx.R = s.R.Bytes()
}
func (s *StandardTx) setTxFee(fee int64) error {
	if fee < 0 {
		return errors.New("fee cannot be negative")
	}
	s.Fee.SetBigInt(big.NewInt(fee))

	s.baseTx.Fee = uint64(fee)

	return nil
}

func (s *StandardTx) AddInput(i *Input) error {
	if len(s.Inputs)+1 > maxInputs {
		return errors.New("maximum amount of inputs reached")
	}
	s.Inputs = append(s.Inputs, i)
	return nil
}

func (s *StandardTx) AddOutput(pubAddr key.PublicAddress, amount ristretto.Scalar) error {
	if len(s.Outputs)+1 > maxOutputs {
		return errors.New("maximum amount of outputs reached")
	}

	pubKey, err := pubAddr.ToKey(s.netPrefix)
	if err != nil {
		return err
	}

	output := NewOutput(s.r, amount, s.index, *pubKey)

	s.Outputs = append(s.Outputs, output)

	s.index = s.index + 1

	s.TotalSent.Add(&s.TotalSent, &amount)

	return nil
}

func (s *StandardTx) ProveRangeProof() error {

	lenOutputs := len(s.Outputs)
	if lenOutputs < 1 {
		return nil
	}

	// Collect all amounts from outputs
	amounts := make([]ristretto.Scalar, 0, lenOutputs)
	for i := 0; i < lenOutputs; i++ {
		amounts = append(amounts, s.Outputs[i].amount)
	}

	// Create range proof
	proof, err := rangeproof.Prove(amounts, false)
	if err != nil {
		return err
	}
	if len(proof.V) != len(amounts) {
		return errors.New("rangeproof did not create proof for all amounts")
	}

	// Move commitment values to their respective outputs
	// along with their blinding factors
	for i := 0; i < lenOutputs; i++ {
		s.Outputs[i].setCommitment(proof.V[i].Value)
		s.Outputs[i].setMask(proof.V[i].BlindingFactor)
	}
	return nil
}

func calculateCommToZero(inputs []*Input, outputs []*Output) {
	var sumOutputMask ristretto.Scalar

	// Aggregate mask values in each outputs commitment
	for _, output := range outputs {
		sumOutputMask.Add(&sumOutputMask, &output.mask)
	}

	// Generate len(input)-1 amount of mask values
	// For the pseudoCommitment
	pseudoMaskValues := generateScalars(len(inputs) - 1)

	// Aggregate all mask values
	var sumPseudoMaskValues ristretto.Scalar
	for i := 0; i < len(pseudoMaskValues); i++ {
		sumPseudoMaskValues.Add(&sumPseudoMaskValues, &pseudoMaskValues[i])
	}

	// Append a new mask value to the array of values
	// s.t. it is equal to sumOutputBlinders - sumInputBlinders
	var lastMaskValue ristretto.Scalar
	lastMaskValue.Sub(&sumOutputMask, &sumPseudoMaskValues)
	pseudoMaskValues = append(pseudoMaskValues, lastMaskValue)

	// Calculate and set the commitment to zero for each input
	for i := range inputs {
		input := inputs[i]
		var commToZero ristretto.Scalar
		commToZero.Sub(&pseudoMaskValues[i], &input.mask)

		input.Proof.SetCommToZero(commToZero)
	}

	// Compute Pseudo commitment for each input
	for i := range inputs {
		input := inputs[i]
		pseduoMask := pseudoMaskValues[i]

		pseudoCommitment := CommitAmount(input.amount, pseduoMask)
		input.setPseudoComm(pseudoCommitment)
	}
}
func (s *StandardTx) AddDecoys(numMixins int, f FetchDecoys) error {

	if f == nil {
		return errors.New("fetch decoys function cannot be nil")
	}

	for _, input := range s.Inputs {
		decoys := f(numMixins)

		keys, offsets, err := decoys.ToMLSAG()
		if err != nil {
			return err
		}

		err = input.AddDecoys(offsets, keys)
		if err != nil {
			return err
		}
	}
	return nil
}

// Prove creates the rangeproof for output values and creates the mlsag balance and ownership proof
// Prove assumes that all inputs, outputs and decoys have been added to the transaction
func (s *StandardTx) Prove() error {

	// Prove rangeproof, creating the commitments for each output
	err := s.ProveRangeProof()
	if err != nil {
		return err
	}

	// Calculate commitment to zero, adding keys to mlsag
	calculateCommToZero(s.Inputs, s.Outputs)

	// Prove Mlsag
	for i := range s.Inputs {

		// Subtract the pseudo commitment from all of the decoy transactions
		input := s.Inputs[i]
		input.Proof.SubCommToZero(input.PseudoCommitment)

		_, _, err = input.Prove()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *StandardTx) Encode() dtx.Standard {
	return s.baseTx
}

func generateScalars(n int) []ristretto.Scalar {

	var scalars []ristretto.Scalar
	for i := 0; i < n; i++ {
		var x ristretto.Scalar
		x.Rand()
		scalars = append(scalars, x)
	}
	return scalars
}

func CommitAmount(amount, mask ristretto.Scalar) ristretto.Point {

	var blindPoint ristretto.Point
	blindPoint.Derive([]byte("blindPoint"))

	var aH, bG, commitment ristretto.Point
	bG.ScalarMultBase(&mask)
	aH.ScalarMult(&blindPoint, &amount)

	commitment.Add(&aH, &bG)

	return commitment
}
