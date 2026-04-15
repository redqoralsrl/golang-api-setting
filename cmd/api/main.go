package main

import (
	"go-template/config"
	"go-template/domain/errorlog"
	errorlogPostgresql "go-template/domain/errorlog/postgresql"
	"go-template/internal/api"
	"go-template/internal/database/postgresql"
	"go-template/internal/jwt"
	"go-template/internal/logger"
	"go-template/internal/logger/zerolog"
	"go-template/internal/validator"

	"go-template/domain/user"
	userPostgresql "go-template/domain/user/postgresql"
	stdHandler "go-template/internal/http/chi"

	_ "github.com/lib/pq"
)

func main() {
	conf := config.LoadConfig()

	stage := conf.Stage
	if stage == "" {
		stage = "dev"
	}

	// setup Logger
	l := zerolog.NewLogger(stage)
	defer func() {
		_ = l.Close()
	}()

	// setup Database
	db, err := postgresql.NewDB(conf, l)
	if err != nil {
		l.Fatal("Failed to connect to database", logger.Field{
			Key:   "error",
			Value: err.Error(),
		})
	}
	defer func() {
		if err = db.Close(); err != nil {
			l.Fatal("Failed to close database", logger.Field{
				Key:   "error",
				Value: err.Error(),
			})
		}
	}()

	// setup Adapter
	transactionManager := postgresql.NewManager(db)
	jwtAdapter := jwt.NewJWTAdapter(conf.APISecretKey)
	// pgAdvisoryLocker := postgresql.NewPgAdvisoryLocker()

	// setup Storage
	errorLogStorage := errorlogPostgresql.NewErrorLog(db)
	userStorage := userPostgresql.NewUser(db)

	// setup Service
	errorLogService := errorlog.NewService(errorLogStorage, l)
	userService := user.NewService(userStorage, jwtAdapter, transactionManager, l)

	if conf.JobsEnabled() {

	} else {
		l.Info("Skipping API cron job registration", logger.Field{Key: "stage", Value: stage})
	}

	// setup handler
	services := stdHandler.Services{
		UserService:     userService,
		ErrorLogService: errorLogService,
	}

	v, err := validator.NewValidationService()
	if err != nil {
		l.Fatal("Failed to initialize validator", logger.Field{
			Key:   "error",
			Value: err,
		})
	}

	s := stdHandler.NewApiRouter(&services, conf, v, jwtAdapter, l)

	err = api.Start(l, conf.APIPort, s)
	if err != nil {
		l.Fatal("Failed to start server", logger.Field{
			Key:   "error",
			Value: err,
		})
	}
}
