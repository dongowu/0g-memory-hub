package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/config"
	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/server"
	"github.com/spf13/cobra"
)

const (
	defaultHTTPReadHeaderTimeout = 5 * time.Second
	defaultHTTPReadTimeout       = 15 * time.Second
	defaultHTTPWriteTimeout      = 30 * time.Second
	defaultHTTPIdleTimeout       = 60 * time.Second
	defaultHTTPShutdownTimeout   = 5 * time.Second
)

type lifecycleHTTPServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the OpenClaw-facing HTTP API",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfg := config.Load()

		svc, closer, err := workflowServiceWithClosableDeps()
		if err != nil {
			return err
		}

		handler := server.NewHandler(svc)
		srv := newHTTPServer(cfg, handler)
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		fmt.Printf("http_addr=%s\n", cfg.HTTPAddr)
		return runHTTPServer(ctx, srv, closer)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: defaultHTTPReadHeaderTimeout,
		ReadTimeout:       defaultHTTPReadTimeout,
		WriteTimeout:      defaultHTTPWriteTimeout,
		IdleTimeout:       defaultHTTPIdleTimeout,
	}
}

func runHTTPServer(ctx context.Context, srv lifecycleHTTPServer, closer io.Closer) error {
	watchCtx, stopWatching := context.WithCancel(ctx)
	defer stopWatching()

	shutdownDone := make(chan error, 1)
	go func() {
		select {
		case <-watchCtx.Done():
			if ctx.Err() == nil {
				return
			}

			shutdownCtx, cancel := context.WithTimeout(context.Background(), defaultHTTPShutdownTimeout)
			defer cancel()

			shutdownErr := srv.Shutdown(shutdownCtx)
			closeErr := closeCloser(closer)

			if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) && closeErr != nil {
				shutdownDone <- errors.Join(shutdownErr, closeErr)
				return
			}
			if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
				shutdownDone <- shutdownErr
				return
			}
			shutdownDone <- closeErr
		}
	}()

	err := srv.ListenAndServe()
	stopWatching()

	if ctx.Err() != nil {
		shutdownErr := <-shutdownDone
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			if shutdownErr != nil {
				return errors.Join(err, shutdownErr)
			}
			return err
		}
		return shutdownErr
	}

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		_ = closeCloser(closer)
		return err
	}

	return closeCloser(closer)
}

func closeCloser(closer io.Closer) error {
	if closer == nil {
		return nil
	}
	return closer.Close()
}
