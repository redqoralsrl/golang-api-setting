package postgresql

import (
	"context"
	"errors"
	"hash/fnv"
	"time"

	"go-template/internal/database"
	"go-template/internal/database/postgresql/gen"
)

func lockParams(ns database.LockNamespace, key string) (int32, int32) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int32(ns), int32(h.Sum32())
}

// PgAdvisoryLocker 트랜잭션 advisory lock. 트랜잭션이 끝나면 자동으로 풀린다.
type PgAdvisoryLocker struct {
	db *Database
}

var _ database.TxLocker = (*PgAdvisoryLocker)(nil)

func NewPgAdvisoryLocker(db *Database) *PgAdvisoryLocker {
	return &PgAdvisoryLocker{db: db}
}

func (l *PgAdvisoryLocker) LockTx(ctx context.Context, ns database.LockNamespace, key string) (bool, error) {
	if _, ok := txFromContext(ctx); !ok {
		return false, errors.New("LockTx must be called inside WithTransaction")
	}
	n, k := lockParams(ns, key)
	return l.db.Queries(ctx).TryAdvisoryXactLock(ctx, gen.TryAdvisoryXactLockParams{Ns: n, Key: k})
}

// PgSessionLocker 풀에서 연결 하나를 따로 잡아 세션 advisory lock 을 건다.
// 서버가 죽어 연결이 끊기면 락도 풀린다.
type PgSessionLocker struct {
	db *Database
}

var _ database.SessionLocker = (*PgSessionLocker)(nil)

func NewPgSessionLocker(db *Database) *PgSessionLocker {
	return &PgSessionLocker{db: db}
}

func (l *PgSessionLocker) TryLock(ctx context.Context, ns database.LockNamespace, key string) (func(), bool, error) {
	conn, err := l.db.pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}

	n, k := lockParams(ns, key)
	queries := gen.New(conn)
	acquired, err := queries.TryAdvisoryLock(ctx, gen.TryAdvisoryLockParams{Ns: n, Key: k})
	if err != nil || !acquired {
		conn.Release()
		return nil, false, err
	}

	release := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		unlocked, err := queries.AdvisoryUnlock(ctx, gen.AdvisoryUnlockParams{Ns: n, Key: k})
		if err != nil || !unlocked {
			// 락을 쥔 연결이 풀로 돌아가면 다음 실행이 영영 락을 못 잡으므로 연결을 닫아 버린다.
			_ = conn.Hijack().Close(ctx)
			return
		}
		conn.Release()
	}
	return release, true, nil
}
