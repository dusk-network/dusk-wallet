package transactions

import (
	"github.com/dusk-network/dusk-wallet/key"
)

type PhoenixNote struct {
    Sk key.PhoenixSecret
    Pos uint64
    Value uint64
    Unspent bool
}
