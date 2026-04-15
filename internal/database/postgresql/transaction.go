package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"go-template/internal/database/postgresql/gen"
)

type transactionKeyType string

const transactionKey transactionKeyType = "dbTx"

func NewContextWithTransaction(ctx context.Context, tx *sql.Tx, cursorInstance *Cursor) context.Context {
	dbTx := &Database{tx, gen.New(tx), cursorInstance}
	return context.WithValue(ctx, transactionKey, dbTx)
}

func TransactionFromContext(ctx context.Context) (*Database, bool) {
	tx, ok := ctx.Value(transactionKey).(*Database)
	return tx, ok
}

type DBTransactionManager interface {
	WithTransaction(ctx context.Context, isolationLevel sql.IsolationLevel, readOnly bool, fn func(ctx context.Context) error) error
}

type Manager struct {
	db     *sql.DB
	cursor *Cursor
}

func NewManager(db *Database) *Manager {
	return &Manager{db: db.Querier.(*sql.DB), cursor: db.cursor}
}

func (m *Manager) WithTransaction(ctx context.Context, isolationLevel sql.IsolationLevel, readOnly bool, fn func(ctx context.Context) error) error {
	opts := &sql.TxOptions{
		Isolation: isolationLevel,
		ReadOnly:  readOnly,
	}

	if tx, ok := TransactionFromContext(ctx); ok {
		if _, ok := tx.Querier.(*sql.Tx); !ok {
			return fmt.Errorf("invalid transaction type %T", tx.Querier)
		}
		return fn(ctx)
	}

	tx, err := m.db.BeginTx(ctx, opts)
	if err != nil {
		return err
	}
	txContext := NewContextWithTransaction(ctx, tx, m.cursor)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			fmt.Println(p)
			//panic(p)
		}
	}()

	if err := fn(txContext); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
