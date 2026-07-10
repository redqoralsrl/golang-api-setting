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
	done    chan struct{}
	logger  logger.Logger
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
				database := &Database{
					Querier: db,
					Queries: gen.New(db),
					cursor:  cursorInstance,
					done:    make(chan struct{}),
					logger:  l,
				}
				database.recordStats()
				go database.observeStats(10 * time.Second)

				return database, nil
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
	close(d.done)
	return d.Querier.(*sql.DB).Close()
}

func (d *Database) observeStats(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			d.recordStats()
		case <-d.done:
			return
		}
	}
}

func (d *Database) recordStats() {
	stats := d.Querier.(*sql.DB).Stats()
	d.logger.Info("Database connection pool stats",
		logger.NewField("idle", stats.Idle),
		logger.NewField("in_use", stats.InUse),
		logger.NewField("open", stats.OpenConnections),
		logger.NewField("wait_count", stats.WaitCount),
		logger.NewField("wait_duration", stats.WaitDuration.String()),
	)
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
