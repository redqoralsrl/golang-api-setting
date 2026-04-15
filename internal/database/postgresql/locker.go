package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"go-template/internal/database"
	"hash/fnv"
)

type PgAdvisoryLocker struct {
}

var _ database.TxLocker = (*PgAdvisoryLocker)(nil)

func NewPgAdvisoryLocker() *PgAdvisoryLocker {
	return &PgAdvisoryLocker{}
}

func (l *PgAdvisoryLocker) LockTx(ctx context.Context, ns database.LockNamespace, key string) (bool, error) {
	db, ok := TransactionFromContext(ctx)
	if !ok {
		return false, fmt.Errorf("no transaction from context")
	}

	tx, ok := db.Querier.(*sql.Tx)
	if !ok {
		return false, fmt.Errorf("querier is not *sql.Tx")
	}

	h := fnv.New32a()
	h.Write([]byte(key))
	keyHash := int32(h.Sum32())

	var acquired bool
	if err := tx.QueryRowContext(ctx,
		`select pg_try_advisory_xact_lock($1,$2)`, ns, keyHash).Scan(&acquired); err != nil {
		return false, err
	}
	return acquired, nil
}
