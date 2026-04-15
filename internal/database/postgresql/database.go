package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"go-template/config"
	"go-template/internal/database/postgresql/gen"
	"go-template/internal/logger"
	"time"

	_ "github.com/lib/pq"
)

const (
	InitialBackoff = 500 * time.Millisecond
	MaxBackoff     = 60 * time.Second
	BackoffFactor  = 2
	MaxRetries     = 10
	MaxIdleConn    = 15
	MaxOpenConn    = 30
)

type Database struct {
	Querier
	Queries *gen.Queries //sqlc로 생성된 쿼리
	cursor  *Cursor
}

type Querier interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func NewDB(config *config.Config, l logger.Logger) (*Database, error) {
	user := config.DBUser
	password := config.DBPassword
	dbname := config.DBName
	host := config.DBHost
	port := config.DBPort

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC", host, port, user, password, dbname)
	if config.Stage != "dev" {
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require timezone=UTC", host, port, user, password, dbname)
	}

	var db *sql.DB
	var err error

	backoff := InitialBackoff
	for i := 0; i < MaxRetries; i++ {
		db, err = sql.Open("postgres", dsn)
		db.SetMaxIdleConns(MaxIdleConn)
		db.SetMaxOpenConns(MaxOpenConn)
		if err == nil {
			err = db.Ping()
			cursorSecret := []byte(config.CursorSecret)
			cursorInstance := NewCursor(cursorSecret)

			if err == nil {
				return &Database{db, gen.New(db), cursorInstance}, nil
			}
		}
		l.Warn("Failed to connect to the database. Retrying...", logger.NewField("error", err), logger.NewField("backoff", backoff))

		time.Sleep(backoff)
		backoff = time.Duration(float64(backoff) * BackoffFactor)
		if backoff > MaxBackoff {
			backoff = MaxBackoff
		}
	}

	l.Error("Failed to connect to the database after multiple retries", logger.NewField("error", err))
	return nil, fmt.Errorf("failed to connect to the database after %d retries: %v", MaxRetries, err)
}

func (d *Database) Close() error {
	return d.Querier.(*sql.DB).Close()
}

func (d *Database) GetQueryRowerFromContext(ctx context.Context) *Database {
	if tx, ok := TransactionFromContext(ctx); ok {
		return tx
	}
	return d
}

func (d *Database) EncryptCursor(id int) (string, error) {
	return d.cursor.Encrypt(id)
}

func (d *Database) DecryptCursor(encodedCursor string) (int, error) {
	return d.cursor.Decrypt(encodedCursor)
}
