package postgresql

import (
	"context"
	"errors"

	"go-template/internal/database"

	"github.com/jackc/pgx/v5"
)

type txKey struct{}

func txFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(pgx.Tx)
	return tx, ok
}

type Manager struct {
	db *Database
}

var _ database.TxManager = (*Manager)(nil)

func NewManager(db *Database) *Manager {
	return &Manager{db: db}
}

var isoLevels = map[database.IsolationLevel]pgx.TxIsoLevel{
	database.ReadCommitted:  pgx.ReadCommitted,
	database.RepeatableRead: pgx.RepeatableRead,
	database.Serializable:   pgx.Serializable,
}

func (m *Manager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error, opts ...database.TxOption) (err error) {
	// 이미 트랜잭션 안이면 바깥 트랜잭션에 합류한다
	if _, ok := txFromContext(ctx); ok {
		return fn(ctx)
	}

	var o database.TxOptions
	for _, opt := range opts {
		opt(&o)
	}
	txOpts := pgx.TxOptions{IsoLevel: isoLevels[o.Isolation], AccessMode: pgx.ReadWrite}
	if o.ReadOnly {
		txOpts.AccessMode = pgx.ReadOnly
	}

	tx, err := m.db.pool.BeginTx(ctx, txOpts)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			// 삼키면 롤백됐는데 nil 이 반환된다. 다시 던져 RecoveryMiddleware 가 500 으로 처리하게 한다.
			panic(p)
		}
	}()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return errors.Join(err, rbErr)
		}
		return err
	}

	return tx.Commit(ctx)
}
