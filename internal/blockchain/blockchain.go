package blockchain

import (
	"context"
	"fmt"
	"go-template/internal/blockchain/erc20"
	"go-template/internal/logger"
	"math/big"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Blockchain struct {
	client *ethclient.Client
	logger logger.Logger
}

type BlockchainAdapter interface {
	GetClient() *ethclient.Client
	GetErc20Balance(ctx context.Context, contract, address string) (*big.Int, error)
}

var _ BlockchainAdapter = (*Blockchain)(nil)

func NewEvmBlockchain(c *ethclient.Client, l logger.Logger) *Blockchain {
	return &Blockchain{
		client: c,
		logger: l,
	}
}

func (b *Blockchain) GetClient() *ethclient.Client {
	return b.client
}

func (b *Blockchain) GetErc20Balance(ctx context.Context, contract, address string) (*big.Int, error) {
	ca, err := parseHexAddress(contract)
	if err != nil {
		return nil, fmt.Errorf("get erc20 balance contract: %w", err)
	}

	ac, err := parseHexAddress(address)
	if err != nil {
		return nil, fmt.Errorf("get erc20 balance account: %w", err)
	}

	erc20Rpc := erc20.NewERC20(b.client, ca.Hex())

	return erc20Rpc.BalanceOf(ctx, ac)
}
