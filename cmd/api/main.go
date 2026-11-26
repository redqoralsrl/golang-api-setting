package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go-template/config"
	"go-template/domain/errorlog"
	errorlogPostgresql "go-template/domain/errorlog/postgresql"
	"go-template/domain/user"
	userPostgresql "go-template/domain/user/postgresql"
	"go-template/internal/api"
	"go-template/internal/database/postgresql"
	stdHandler "go-template/internal/http/chi"
	"go-template/internal/jobs"
	"go-template/internal/jwt"
	"go-template/internal/logger"
	"go-template/internal/logger/zerolog"
	"go-template/internal/validator"
	"go-template/migrations"
)

// version 빌드 시 -ldflags "-X main.version=..." 로 넣는다 (Dockerfile.prod)
var version = "dev"

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

// run 종료 순서: 신호 수신 → HTTP 요청 마무리 → 주기 작업 중단 → DB 연결 종료.
// defer 가 역순으로 실행되므로 만든 순서의 반대로 정리된다.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conf, confErr := config.Load()

	// setup Logger
	l := zerolog.NewLogger(string(conf.Stage))
	defer func() { _ = l.Close() }()
	if confErr != nil {
		l.Error("Invalid configuration", logger.NewError(confErr))
		return confErr
	}
	l.Info("Starting API", logger.NewField("version", version), logger.NewField("stage", conf.Stage))

	// setup Database
	if conf.MigrateOnStart {
		if err := migrate(ctx, conf, l); err != nil {
			l.Error("Failed to migrate database", logger.NewError(err))
			return err
		}
	}
	db, err := postgresql.NewDB(ctx, conf, l)
	if err != nil {
		l.Error("Failed to connect to database", logger.NewError(err))
		return err
	}
	defer db.Close()

	// setup Adapter
	transactionManager := postgresql.NewManager(db)
	jwtAdapter, err := jwt.NewJWTAdapter(conf.APISecretKey)
	if err != nil {
		l.Error("Failed to initialize jwt adapter", logger.NewError(err))
		return err
	}

	// setup Storage
	errorLogStorage := errorlogPostgresql.NewErrorLog(db)
	userStorage := userPostgresql.NewUser(db)

	// setup Service
	errorLogService := errorlog.NewService(errorLogStorage, l)
	userService := user.NewService(userStorage, jwtAdapter, transactionManager, l)

	// setup Jobs (JOBS_ENABLED=true 일 때만). 서버가 여러 대여도 작업마다 한 대만 실행한다.
	if conf.JobsEnabled {
		scheduler := jobs.NewScheduler(postgresql.NewPgSessionLocker(db), l)
		// 예시:
		// if err := scheduler.Add(jobs.Job{
		// 	Name:    "user.cleanup",
		// 	Spec:    "0 * * * *", // 매시 정각 (UTC)
		// 	Timeout: 5 * time.Minute,
		// 	Run:     userService.Cleanup,
		// }); err != nil {
		// 	l.Error("Failed to register job", logger.NewError(err))
		// 	return err
		// }
		scheduler.Start()
		defer scheduler.Stop()
	}

	// setup handler
	services := stdHandler.Services{
		UserService:     userService,
		ErrorLogService: errorLogService,
	}

	v, err := validator.NewValidationService()
	if err != nil {
		l.Error("Failed to initialize validator", logger.NewError(err))
		return err
	}

	router := stdHandler.NewApiRouter(&services, conf, v, jwtAdapter, l, db.Ping)

	if err := api.Serve(ctx, l, conf.APIPort, router); err != nil {
		l.Error("Server stopped with error", logger.NewError(err))
		return err
	}
	return nil
}

func migrate(ctx context.Context, conf *config.Config, l logger.Logger) error {
	m, err := postgresql.NewMigrator(conf.DB, migrations.FS, l)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	results, err := m.Up(ctx)
	for _, r := range results {
		l.Info("Applied migration", logger.NewField("file", r.Source.Path), logger.NewField("duration", r.Duration.String()))
	}
	return err
}
