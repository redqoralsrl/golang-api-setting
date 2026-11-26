package main

import (
	"go-template/config"
	"go-template/domain/errorlog"
	errorlogPostgresql "go-template/domain/errorlog/postgresql"
	"go-template/domain/user"
	userPostgresql "go-template/domain/user/postgresql"
	"go-template/internal/api"
	"go-template/internal/database/postgresql"
	stdHandler "go-template/internal/http/chi"
	"go-template/internal/jwt"
	"go-template/internal/logger"
	"go-template/internal/logger/zerolog"
	"go-template/internal/validator"

	// gRPC 미사용. 활성화 방법은 GRPC.md
	// usergrpc "go-template/internal/grpc/user"
	// userv1 "go-template/proto/user/v1"
	// "google.golang.org/grpc"

	_ "github.com/lib/pq"
)

func main() {
	conf, confErr := config.Load()

	// setup Logger
	l := zerolog.NewLogger(string(conf.Stage))
	defer func() {
		_ = l.Close()
	}()
	if confErr != nil {
		l.Fatal("Invalid configuration", logger.NewError(confErr))
	}

	// setup Database
	db, err := postgresql.NewDB(conf, l)
	if err != nil {
		l.Fatal("Failed to connect to database", logger.NewError(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			l.Error("Failed to close database", logger.NewError(err))
		}
	}()

	// setup Adapter
	transactionManager := postgresql.NewManager(db)
	jwtAdapter, err := jwt.NewJWTAdapter(conf.APISecretKey)
	if err != nil {
		l.Fatal("Failed to initialize jwt adapter", logger.NewError(err))
	}

	// setup Storage
	errorLogStorage := errorlogPostgresql.NewErrorLog(db)
	userStorage := userPostgresql.NewUser(db)

	// setup Service
	errorLogService := errorlog.NewService(errorLogStorage, l)
	userService := user.NewService(userStorage, jwtAdapter, transactionManager, l)

	// setup handler
	services := stdHandler.Services{
		UserService:     userService,
		ErrorLogService: errorLogService,
	}

	v, err := validator.NewValidationService()
	if err != nil {
		l.Fatal("Failed to initialize validator", logger.NewError(err))
	}

	s := stdHandler.NewApiRouter(&services, conf, v, jwtAdapter, l, db.Ping)

	// gRPC 사용 시 주석 해제 (GRPC.md 참고)
	// go func() {
	// 	err := api.StartGRPC(l, conf.GRPCPort, func(grpcServer *grpc.Server) {
	// 		userv1.RegisterUserServiceServer(grpcServer, usergrpc.NewServer(userService))
	// 	})
	// 	if err != nil {
	// 		l.Fatal("Failed to start gRPC server", logger.NewError(err))
	// 	}
	// }()

	if err := api.Start(l, conf.APIPort, s); err != nil {
		l.Error("Server stopped with error", logger.NewError(err))
	}
}
