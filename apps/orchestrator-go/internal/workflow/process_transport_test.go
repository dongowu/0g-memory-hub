package workflow

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProcessTransportCallSuccess(t *testing.T) {
	t.Parallel()

	transport := NewProcessTransport(
		os.Args[0],
		"-test.run=TestProcessTransportHelper",
		"--",
	)
	transport.SetEnv([]string{
		"GO_WANT_HELPER_PROCESS=1",
		"HELPER_MODE=success",
	})

	resp, err := transport.Call(context.Background(), []byte(`{"cmd":"ping"}`))
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	got := strings.TrimSpace(string(resp))
	want := `{"kind":"state","state":{"workflow_id":"wf-helper","agent_id":"agent-helper","status":"Running","latest_step":0,"latest_root":"","events":[]}}`
	if got != want {
		t.Fatalf("response mismatch\ngot:  %s\nwant: %s", got, want)
	}
}

func TestProcessTransportCallProcessFailure(t *testing.T) {
	t.Parallel()

	transport := NewProcessTransport(
		os.Args[0],
		"-test.run=TestProcessTransportHelper",
		"--",
	)
	transport.SetEnv([]string{
		"GO_WANT_HELPER_PROCESS=1",
		"HELPER_MODE=fail",
	})

	_, err := transport.Call(context.Background(), []byte(`{"cmd":"ping"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "helper failed intentionally") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessTransportCallContextTimeout(t *testing.T) {
	t.Parallel()

	transport := NewProcessTransport(
		os.Args[0],
		"-test.run=TestProcessTransportHelper",
		"--",
	)
	transport.SetEnv([]string{
		"GO_WANT_HELPER_PROCESS=1",
		"HELPER_MODE=sleep",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	_, err := transport.Call(ctx, []byte(`{"cmd":"ping"}`))
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if ctx.Err() == nil {
		t.Fatalf("expected context error, got: %v", err)
	}
}

func TestProcessTransportHelper(t *testing.T) {
	t.Helper()

	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mode := os.Getenv("HELPER_MODE")
	switch mode {
	case "success":
		_, _ = fmt.Fprintln(os.Stdout, `{"kind":"state","state":{"workflow_id":"wf-helper","agent_id":"agent-helper","status":"Running","latest_step":0,"latest_root":"","events":[]}}`)
		os.Exit(0)
	case "fail":
		_, _ = fmt.Fprintln(os.Stderr, "helper failed intentionally")
		os.Exit(12)
	case "sleep":
		time.Sleep(500 * time.Millisecond)
		_, _ = fmt.Fprintln(os.Stdout, `{"kind":"state","state":{"workflow_id":"wf-sleep","agent_id":"agent-sleep","status":"Running","latest_step":0,"latest_root":"","events":[]}}`)
		os.Exit(0)
	default:
		_, _ = fmt.Fprintf(os.Stderr, "unknown helper mode: %s\n", mode)
		os.Exit(13)
	}
}
