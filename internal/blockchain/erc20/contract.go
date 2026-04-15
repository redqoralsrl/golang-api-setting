package erc20

import (
	"context"
	"go-template/internal/blockchain/erc20/solidity"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ERC20 struct {
	client   *ethclient.Client
	instance *solidity.SolidityCaller
	address  string
}

var _ Rpc = (*ERC20)(nil)

func NewERC20(c *ethclient.Client, address string) *ERC20 {
	addr := common.HexToAddress(address)
	instance, _ := solidity.NewSolidityCaller(addr, c)

	return &ERC20{
		client:   c,
		instance: instance,
		address:  address,
	}
}

func (e *ERC20) GetAddress() common.Address {
	return common.HexToAddress(e.address)
}

func (e *ERC20) BalanceOf(ctx context.Context, account common.Address) (*big.Int, error) {
	balance, err := e.instance.BalanceOf(&bind.CallOpts{
		Context: ctx,
	}, account)
	if err != nil {
		return nil, err
	}

	return balance, nil
}

func (e *ERC20) Name(ctx context.Context) (string, error) {
	name, err := e.instance.Name(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return "", err
	}

	return name, nil
}

func (e *ERC20) Symbol(ctx context.Context) (string, error) {
	symbol, err := e.instance.Symbol(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return "", err
	}

	return symbol, nil
}

func (e *ERC20) Decimals(ctx context.Context) (uint8, error) {
	decimals, err := e.instance.Decimals(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return 0, err
	}

	return decimals, nil
}

func (e *ERC20) TotalSupply(ctx context.Context) (*big.Int, error) {
	totalSupply, err := e.instance.TotalSupply(&bind.CallOpts{
		Context: ctx,
	})
	if err != nil {
		return nil, err
	}

	return totalSupply, nil
}
