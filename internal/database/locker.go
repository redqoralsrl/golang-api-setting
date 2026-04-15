package database

import "context"

type TxLocker interface {
	LockTx(ctx context.Context, ns LockNamespace, key string) (bool, error)
}

type LockNamespace int32

const (
	NSLockName  LockNamespace = 1001
)
