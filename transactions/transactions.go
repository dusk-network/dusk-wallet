package transactions

import (
	"dusk-wallet/key"
	"dusk-wallet/mlsag"
	"errors"
	"math/big"

	ristretto "github.com/bwesterb/go-ristretto"
)

const maxInputs = 2000
const maxOutputs = 32

// FetchDecoys returns a slice of decoy pubKey vectors
type FetchDecoys func(numMixins int, numKeysPerUser int) []mlsag.PubKeys

type StealthTx struct {
	r ristretto.Scalar
	R ristretto.Point

	Inputs  []*Input
	Outputs []*Output
	Fee     ristretto.Scalar

	index     uint32
	netPrefix byte

	TotalSent ristretto.Scalar
}

func New(netPrefix byte, fee int64) (*StealthTx, error) {

	tx := &StealthTx{}

	tx.TotalSent.SetZero()

	// Index for subaddresses
	tx.index = 0

	// prefix to signify testnet/
	tx.netPrefix = netPrefix

	// randomly generated nonce - r
	var r ristretto.Scalar
	r.Rand()
	tx.r = r

	tx.R.ScalarMultBase(&r)

	if fee < 0 {
		return nil, errors.New("fee cannot be negative")
	}
	tx.Fee.SetBigInt(big.NewInt(fee))

	return tx, nil
}

func (s *StealthTx) AddInput(i *Input) error {
	if len(s.Inputs)+1 > maxInputs {
		return errors.New("maximum amount of inputs reached")
	}
	s.Inputs = append(s.Inputs, i)
	return nil
}

func (s *StealthTx) CalcCommToZero() error {
	var aggOutputBlinders ristretto.Scalar
	aggOutputBlinders.SetZero()

	for _, output := range s.Outputs {
		aggOutputBlinders.Add(&aggOutputBlinders, &output.mask)
	}

	var aggInputBlinders ristretto.Scalar
	aggInputBlinders.SetZero()
	for index, input := range s.Inputs {

		var pseudoMask ristretto.Scalar
		pseudoMask.Rand()

		// Blinder for last item is Sum(previous input blinders) - Sum(output blinders)
		if index == len(s.Inputs)-1 {
			pseudoMask.Sub(&aggOutputBlinders, &aggInputBlinders)
		}
		pseudoCommitment := CommitAmount(input.amount, pseudoMask)

		input.pseudoMask.Set(&pseudoMask)
		input.PseudoCommitment.Set(&pseudoCommitment)

		var commToZero ristretto.Scalar
		commToZero.Sub(&input.pseudoMask, &input.mask)

		privKeys := mlsag.PrivKeys{}

		// Add key for pubkey to unlock input
		privKeys.AddPrivateKey(input.privKey)

		// Add commitment to zero
		privKeys.AddPrivateKey(commToZero)

		// Add key vector to mlsag
		input.AddSecretKeyVector(privKeys)

		// Assume decoys have been added already

		// Compute proof
		err := input.Prove()
		if err != nil {
			return err
		}

		// aggregate pseudoOut blinders
		//XXX: This is unecessary for last input
		aggInputBlinders.Add(&aggInputBlinders, &pseudoMask)
	}
	return nil
}

func (s *StealthTx) AddOutput(pubAddr key.PublicAddress, amount ristretto.Scalar) error {
	if len(s.Outputs)+1 > maxOutputs {
		return errors.New("maximum amount of outputs reached")
	}

	pubKey, err := pubAddr.ToKey(s.netPrefix)
	if err != nil {
		return err
	}

	output, err := NewOutput(s.r, amount, s.index, *pubKey)
	if err != nil {
		return err
	}
	s.Outputs = append(s.Outputs, output)

	s.index = s.index + 1

	s.TotalSent.Add(&s.TotalSent, &amount)

	return nil
}

func (s *StealthTx) AddDecoys(numMixins int, f FetchDecoys) error {

	numKeysPerUser := 2
	for _, input := range s.Inputs {
		decoys := f(numMixins, numKeysPerUser)
		input.Proof.AddDecoys(decoys)
	}
	return nil
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
