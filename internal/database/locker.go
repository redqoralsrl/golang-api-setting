package database

import "context"

// TxLocker 트랜잭션이 끝날 때 자동으로 풀리는 락. ctx 는 TxManager.WithTransaction 안의 ctx 여야 한다.
type TxLocker interface {
	LockTx(ctx context.Context, ns LockNamespace, key string) (acquired bool, err error)
}

// SessionLocker 트랜잭션과 무관하게 release 를 부를 때까지 락을 쥔다. 서버가 죽으면 연결과 함께 풀린다.
type SessionLocker interface {
	TryLock(ctx context.Context, ns LockNamespace, key string) (release func(), acquired bool, err error)
}

// LockNamespace 같은 key 문자열이라도 용도별로 락이 겹치지 않게 나눈다.
type LockNamespace int32

const (
	NSLockName LockNamespace = 1001
	NSJob      LockNamespace = 1002 // internal/jobs 스케줄러
)
