package api

import (
	"fmt"
	"net"

	"go-template/internal/logger"

	"google.golang.org/grpc"
)

func StartGRPC(l logger.Logger, port string, register func(*grpc.Server)) error {
	/**
	TODO: 보안 적용 및 metadata(jwt) 적용 필수
	creds, err := credentials.NewServerTLSFromFile("server.crt", "server.key")
	if err != nil {
		return err
	}

	server := grpc.NewServer(
		grpc.Creds(creds),
	)
	*/
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	register(server)

	l.Info(fmt.Sprintf("gRPC listening on port %s", port))

	return server.Serve(listener)
}
