package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"go-template/config"
	"go-template/internal/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Migrator goose 마이그레이션 실행기. 앱 풀과 따로 연결을 열어 statement_timeout 없이 실행한다.
// 서버 여러 대가 동시에 시작해도 advisory lock 으로 한 대만 적용하고 나머지는 기다린다.
type Migrator struct {
	db       *sql.DB
	provider *goose.Provider
}

func NewMigrator(d config.Database, migrations fs.FS, l logger.Logger) (*Migrator, error) {
	connConf, err := pgx.ParseConfig(DSN(d))
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	connConf.RuntimeParams["timezone"] = "UTC"
	db := stdlib.OpenDB(*connConf)

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations,
		goose.WithSessionLocker(locker),
		goose.WithLogger(gooseLogger{l}),
	)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create migrator: %w", err)
	}
	return &Migrator{db: db, provider: provider}, nil
}

// Up 아직 적용하지 않은 마이그레이션을 모두 적용한다.
func (m *Migrator) Up(ctx context.Context) ([]*goose.MigrationResult, error) {
	return m.provider.Up(ctx)
}

// Down 마지막 마이그레이션 하나를 되돌린다.
func (m *Migrator) Down(ctx context.Context) (*goose.MigrationResult, error) {
	return m.provider.Down(ctx)
}

func (m *Migrator) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	return m.provider.Status(ctx)
}

func (m *Migrator) Close() error {
	return m.provider.Close()
}

type gooseLogger struct{ l logger.Logger }

func (g gooseLogger) Printf(format string, v ...any) { g.l.Info(fmt.Sprintf(format, v...)) }
func (g gooseLogger) Fatalf(format string, v ...any) { g.l.Fatal(fmt.Sprintf(format, v...)) }
