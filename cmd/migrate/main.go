// DB 마이그레이션 CLI. Makefile 의 migrate-* 타깃이 부른다.
//
//	go run ./cmd/migrate up            # 아직 안 된 것 전부 적용
//	go run ./cmd/migrate down          # 마지막 1개 되돌리기
//	go run ./cmd/migrate status        # 적용 현황
//	go run ./cmd/migrate create <name> # migrations/<시각>_<name>.sql 생성 (DB 불필요)
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"text/tabwriter"
	"time"

	"go-template/config"
	"go-template/internal/database/postgresql"
	"go-template/internal/logger/zerolog"
	"go-template/migrations"

	"github.com/pressly/goose/v3"
)

const migrationsDir = "migrations"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate <up|down|status|create NAME>")
	os.Exit(2)
}

func run(command string, args []string) error {
	if command == "create" {
		if len(args) != 1 {
			usage()
		}
		return goose.Create(nil, migrationsDir, args[0], "sql")
	}

	dbConf, err := config.LoadDatabase()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	m, err := postgresql.NewMigrator(dbConf, migrations.FS, zerolog.NewLogger("dev"))
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	switch command {
	case "up":
		results, err := m.Up(ctx)
		for _, r := range results {
			fmt.Printf("applied  %s (%s)\n", r.Source.Path, r.Duration.Round(time.Millisecond))
		}
		if err == nil && len(results) == 0 {
			fmt.Println("no pending migrations")
		}
		return err
	case "down":
		r, err := m.Down(ctx)
		if r != nil {
			fmt.Printf("reverted %s (%s)\n", r.Source.Path, r.Duration.Round(time.Millisecond))
		}
		return err
	case "status":
		statuses, err := m.Status(ctx)
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "STATE\tAPPLIED AT\tFILE")
		for _, s := range statuses {
			applied := "-"
			if !s.AppliedAt.IsZero() {
				applied = s.AppliedAt.UTC().Format(time.DateTime)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", s.State, applied, s.Source.Path)
		}
		return w.Flush()
	default:
		usage()
		return nil
	}
}
