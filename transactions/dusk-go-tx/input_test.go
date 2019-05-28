package transactions_test

import (
	"bytes"
	"testing"

	transactions "dusk-wallet/transactions/dusk-go-tx"

	"github.com/stretchr/testify/assert"
	helper "gitlab.dusk.network/dusk-core/dusk-go/pkg/core/tests/helper"
)

func TestEncodeDecodeInput(t *testing.T) {

	assert := assert.New(t)

	// Random input
	keyImage := helper.RandomSlice(t, 32)
	in, err := transactions.NewInput(keyImage)
	for i := 0; i < 100; i++ {
		in.AddInput(helper.RandomSlice(t, 32))
	}
	assert.Nil(err)

	// Encode random input into buffer
	buf := new(bytes.Buffer)
	err = in.Encode(buf)
	assert.Nil(err)

	// Decode buffer into a new input struct
	decIn := &transactions.Input{}
	err = decIn.Decode(buf)
	assert.Nil(err)

	// Decoded input should equal original
	assert.True(decIn.Equals(in))
}
func TestMalformedInput(t *testing.T) {

	// random malformed input
	// should return an error and nil input object
	in, err := helper.RandomInput(t, true)
	assert.Nil(t, in)
	assert.NotNil(t, err)
}
