package api

import (
	"context"
	"errors"
	"fmt"
	"go-template/internal/logger"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

const TIMEOUT = 30 * time.Second

// Graceful shutdown
func Start(l logger.Logger, port string, handler http.Handler) error {
	srv := &http.Server{
		ReadTimeout:       TIMEOUT,
		WriteTimeout:      TIMEOUT,
		Addr:              ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	)
	defer stop()
	errShutdown := make(chan error, 1)
	go shutdown(srv, ctx, errShutdown)

	l.Info(fmt.Sprintf("listening on port %s\n", port))
	err := srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	err = <-errShutdown
	if err != nil {
		return err
	}
	return nil
}

func shutdown(server *http.Server, ctxShutdown context.Context, errShutdown chan error) {
	<-ctxShutdown.Done()

	ctxTimeout, stop := context.WithTimeout(context.Background(), TIMEOUT)
	defer stop()

	err := server.Shutdown(ctxTimeout)
	switch {
	case err == nil:
		errShutdown <- nil
	case errors.Is(err, context.DeadlineExceeded):
		errShutdown <- fmt.Errorf("forcing closing the server")
	default:
		errShutdown <- fmt.Errorf("forcing closing the server")
	}
}
