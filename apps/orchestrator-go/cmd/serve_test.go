package cmd

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/config"
)

func TestRootCommandIncludesServeCommand(t *testing.T) {
	t.Parallel()

	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "serve" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("serve command not registered on root command")
	}
}

func TestNewRuntimeTransportUsesPersistentProcessTransport(t *testing.T) {
	t.Parallel()

	transport := newRuntimeTransport(config.Config{RuntimeBinaryPath: "/tmp/memory-core-rpc"})
	if transport == nil {
		t.Fatal("newRuntimeTransport() returned nil")
	}
}

func TestNewHTTPServerConfiguresTimeouts(t *testing.T) {
	t.Parallel()

	srv := newHTTPServer(config.Config{HTTPAddr: "127.0.0.1:18080"}, http.NewServeMux())
	if srv.Addr != "127.0.0.1:18080" {
		t.Fatalf("Addr = %s, want 127.0.0.1:18080", srv.Addr)
	}
	if srv.ReadHeaderTimeout <= 0 || srv.ReadTimeout <= 0 || srv.WriteTimeout <= 0 || srv.IdleTimeout <= 0 {
		t.Fatalf("expected positive HTTP timeouts, got readHeader=%s read=%s write=%s idle=%s",
			srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout)
	}
}

func TestRunHTTPServerShutdownClosesRuntimeTransport(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	server := &fakeHTTPServer{
		listenErr: http.ErrServerClosed,
		release:   make(chan struct{}),
	}
	closer := &fakeCloser{}

	done := make(chan error, 1)
	go func() {
		done <- runHTTPServer(ctx, server, closer)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runHTTPServer() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("runHTTPServer() did not return after cancellation")
	}

	if !server.shutdownCalled {
		t.Fatal("expected Shutdown() to be called")
	}
	if !closer.closed {
		t.Fatal("expected runtime transport closer to be closed")
	}
}

type fakeHTTPServer struct {
	listenErr      error
	release        chan struct{}
	shutdownCalled bool
}

func (f *fakeHTTPServer) ListenAndServe() error {
	<-f.release
	return f.listenErr
}

func (f *fakeHTTPServer) Shutdown(_ context.Context) error {
	f.shutdownCalled = true
	select {
	case <-f.release:
	default:
		close(f.release)
	}
	return nil
}

type fakeCloser struct {
	closed bool
}

func (f *fakeCloser) Close() error {
	f.closed = true
	return nil
}

var _ io.Closer = (*fakeCloser)(nil)
