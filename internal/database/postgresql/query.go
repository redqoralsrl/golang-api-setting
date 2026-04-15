package postgresql

import (
	"database/sql"
)

type Query interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}
