package postgresql

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"go-template/config"
	"go-template/internal/database/postgresql/gen"
	"go-template/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 60 * time.Second
	maxRetries     = 10

	maxConns        = 30
	minIdleConns    = 2
	maxConnIdleTime = 5 * time.Minute
	maxConnLifetime = time.Hour
)

// sessionParams 앱 연결마다 거는 세션 설정. 마이그레이션 연결에는 걸지 않는다(오래 걸리는 DDL 이 끊기지 않게).
var sessionParams = map[string]string{
	"timezone":                            "UTC",
	"statement_timeout":                   "10s",
	"lock_timeout":                        "5s",
	"idle_in_transaction_session_timeout": "30s",
}

type Database struct {
	pool    *pgxpool.Pool
	queries *gen.Queries
	cursor  *Cursor
}

// NewDB 연결 풀을 만들고 Ping 이 될 때까지 지수 백오프(0.5s → 60s, 최대 10회)로 재시도한다.
func NewDB(ctx context.Context, conf *config.Config, l logger.Logger) (*Database, error) {
	cursor, err := NewCursor([]byte(conf.CursorSecret))
	if err != nil {
		return nil, err
	}

	poolConf, err := pgxpool.ParseConfig(DSN(conf.DB))
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	for k, v := range sessionParams {
		poolConf.ConnConfig.RuntimeParams[k] = v
	}
	poolConf.MaxConns = maxConns
	poolConf.MinIdleConns = minIdleConns
	poolConf.MaxConnIdleTime = maxConnIdleTime
	poolConf.MaxConnLifetime = maxConnLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolConf)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}

	backoff := initialBackoff
	for attempt := 1; ; attempt++ {
		if err = pool.Ping(ctx); err == nil {
			return &Database{pool: pool, queries: gen.New(pool), cursor: cursor}, nil
		}
		if attempt == maxRetries {
			break
		}
		l.Warn("Failed to connect to the database. Retrying...", logger.NewError(err), logger.NewField("backoff", backoff.String()))
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, maxBackoff)
	}

	pool.Close()
	return nil, fmt.Errorf("connect to database after %d attempts: %w", maxRetries, err)
}

// DSN postgres URL. 비밀번호에 특수문자가 있어도 안전하게 인코딩한다.
func DSN(d config.Database) string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, d.Port),
		Path:     d.Name,
		RawQuery: url.Values{"sslmode": {d.SSLMode}}.Encode(),
	}
	return u.String()
}

func (d *Database) Close() { d.pool.Close() }

func (d *Database) Ping(ctx context.Context) error { return d.pool.Ping(ctx) }

// Queries sqlc 쿼리. ctx 에 트랜잭션이 있으면(TxManager.WithTransaction 안) 그 트랜잭션으로 실행한다.
func (d *Database) Queries(ctx context.Context) *gen.Queries {
	if tx, ok := txFromContext(ctx); ok {
		return d.queries.WithTx(tx)
	}
	return d.queries
}

func (d *Database) EncryptCursor(id int64) (string, error) { return d.cursor.Encrypt(id) }

func (d *Database) DecryptCursor(encoded string) (int64, error) { return d.cursor.Decrypt(encoded) }
