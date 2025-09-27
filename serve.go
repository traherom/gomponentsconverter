package gomponentsconverter

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Serve(ctx context.Context, port int) error {
	logger := slog.Default()

	r := BuildRoutes(ctx)

	httpServer := http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           r,
		BaseContext:       func(l net.Listener) context.Context { return ctx },
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
	}

	logger.Info("Starting listeners", "httpPort", port)

	srvErrs := make(chan error, 1)
	go func() {
		srvErrs <- httpServer.ListenAndServe()
	}()

	shutdownHTTP := gracefulShutdown(logger, &httpServer)
	exitCtx, _ := signalHandler(ctx, 60*time.Second)

	select {
	case err := <-srvErrs:
		shutdownHTTP(err)
	case <-exitCtx.Done():
		shutdownHTTP(exitCtx.Err())
	}

	logger.Info("Server exiting")
	return nil
}

func gracefulShutdown(logger *slog.Logger, srv *http.Server) func(reason any) {
	return func(reason any) {
		logger.Info("Server Shutdown", "reason", reason)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error("Error Gracefully Shutting Down API", "error", err)
		}
	}
}

func signalHandler(higherCtx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	logger := slog.Default()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	signalCtx, signalCancel := signal.NotifyContext(higherCtx, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-quit:
			logger.Warn("Initial interrupt signal detected, starting timeout", "timeout", timeout)
		case <-signalCtx.Done():
			logger.Info("Children requested to exit, force quit after timeout", "timeout", timeout)
		case <-higherCtx.Done():
			logger.Debug("Clean exit")
			return
		}

		timer := time.After(timeout)
		select {
		case <-quit:
			logger.Info("Second interrupt signal detected")
		case <-timer:
			logger.Info("Exit timed out", "timeout", timeout)
		case <-higherCtx.Done():
			logger.Info("Clean exit prior to timeout")
			return
		}

		logger.Error("Clean exit not detected, killing everything")
		os.Exit(1)
	}()

	return signalCtx, signalCancel
}
