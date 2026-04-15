package account

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"go-template/internal/logger"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Account struct {
	client *ethclient.Client
	pk     *ecdsa.PrivateKey
	logger logger.Logger
}

var _ AccountAdapter = (*Account)(nil)

type AccountAdapter interface {
	GetAddress() (common.Address, error)
	GetCoinBalance(ctx context.Context, address common.Address) (*big.Int, error)
	HasPendingTx(ctx context.Context, address common.Address) (hasPending bool, latestNonce uint64, pendingNonce uint64, err error)
	GetTransactSetup(ctx context.Context) (auth *bind.TransactOpts, address common.Address, nonce uint64, gasPrice *big.Int, err error)
}

func NewAccount(c *ethclient.Client, pk *ecdsa.PrivateKey, logger logger.Logger) *Account {
	return &Account{
		client: c,
		pk:     pk,
		logger: logger,
	}
}

func (a *Account) GetAddress() (common.Address, error) {
	pubKey := a.pk.Public()
	publicKeyECDSA, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, fmt.Errorf("invalid public key type: %T", pubKey)
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address, nil
}

func (a *Account) GetCoinBalance(ctx context.Context, address common.Address) (*big.Int, error) {
	balance, err := a.client.BalanceAt(ctx, address, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get coin balance: %w", err)
	}

	return balance, nil
}

func (a *Account) HasPendingTx(ctx context.Context, address common.Address) (hasPending bool, latestNonce uint64, pendingNonce uint64, err error) {
	latestNonce, err = a.client.NonceAt(ctx, address, nil)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to get latest nonce: %w", err)
	}

	pendingNonce, err = a.client.PendingNonceAt(ctx, address)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to get pending nonce: %w", err)
	}

	return pendingNonce > latestNonce, latestNonce, pendingNonce, nil
}

func (a *Account) GetTransactSetup(ctx context.Context) (auth *bind.TransactOpts, address common.Address, nonce uint64, gasPrice *big.Int, err error) {
	chainID, err := a.client.ChainID(ctx)
	if err != nil {
		a.logger.Error("failed to get chain id from client", logger.Field{
			Key:   "error",
			Value: err,
		})
		return
	}

	auth, err = bind.NewKeyedTransactorWithChainID(a.pk, chainID)
	if err != nil {
		a.logger.Error("failed to get transactor", logger.Field{
			Key:   "error",
			Value: err,
		})
		return
	}

	address = auth.From

	nonce, err = a.client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		a.logger.Error("failed to get pending nonce from client", logger.Field{
			Key:   "error",
			Value: err,
		})
		return
	}

	gasPrice, err = a.client.SuggestGasPrice(ctx)
	if err != nil {
		a.logger.Error("failed to get gas price", logger.Field{
			Key:   "error",
			Value: err,
		})
		return
	}

	return
}
