package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go-template/internal/logger"
)

const (
	timeout         = 30 * time.Second
	shutdownTimeout = 30 * time.Second
)

// Serve ctx 가 끝날 때까지 HTTP 서버를 돌리고, 끝나면 진행 중인 요청을 기다려 종료한다(graceful shutdown).
func Serve(ctx context.Context, l logger.Logger, port string, handler http.Handler) error {
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           handler,
		ReadTimeout:       timeout,
		WriteTimeout:      timeout,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		l.Info(fmt.Sprintf("HTTP listening on port %s", port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	l.Info("Shutting down HTTP server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
