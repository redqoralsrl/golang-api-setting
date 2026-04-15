package erc20

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Caller interface {
	GetAddress() common.Address
	BalanceOf(ctx context.Context, account common.Address) (*big.Int, error)
	Name(ctx context.Context) (string, error)
	Symbol(ctx context.Context) (string, error)
	Decimals(ctx context.Context) (uint8, error)
	TotalSupply(ctx context.Context) (*big.Int, error)
}

type Transact interface {
}

type Event interface {
}

type Rpc interface {
	Caller
	Transact
	Event
}
