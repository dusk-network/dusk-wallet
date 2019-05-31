package transactions

import dtx "dusk-wallet/transactions/dusk-go-tx"

type TimeLockTx struct {
	lockTime uint64
	*StandardTx
}

func NewTimeLockTx(netPrefix byte, fee int64, lock uint64) (*TimeLockTx, error) {
	standard, err := NewStandard(netPrefix, fee)
	if err != nil {
		return nil, err
	}
	return &TimeLockTx{
		lockTime:   lock,
		StandardTx: standard,
	}, nil
}

func (tl *TimeLockTx) Encode() (dtx.TimeLock, error) {

	baseTimeLockTx := dtx.TimeLock{}
	baseTimeLockTx.Lock = tl.lockTime

	standard, err := tl.StandardTx.Encode()
	if err != nil {
		return dtx.TimeLock{}, err
	}
	baseTimeLockTx.Standard = standard

	return baseTimeLockTx, nil
}
