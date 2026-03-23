package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPersistentProcessTransportReusesSingleProcess(t *testing.T) {
	t.Parallel()

	transport := NewPersistentProcessTransport(
		os.Args[0],
		"-test.run=TestPersistentProcessTransportHelper",
		"--",
	)
	transport.SetEnv([]string{
		"GO_WANT_PERSISTENT_HELPER_PROCESS=1",
		"PERSISTENT_HELPER_MODE=loop_success",
	})

	first, err := transport.Call(context.Background(), []byte(`{"cmd":"ping-1"}`))
	if err != nil {
		t.Fatalf("first Call() error = %v", err)
	}
	second, err := transport.Call(context.Background(), []byte(`{"cmd":"ping-2"}`))
	if err != nil {
		t.Fatalf("second Call() error = %v", err)
	}
	defer func() { _ = transport.Close() }()

	pid1 := responseAgentID(t, first)
	pid2 := responseAgentID(t, second)
	if pid1 != pid2 {
		t.Fatalf("expected same helper process, got %s then %s", pid1, pid2)
	}
	if !strings.Contains(string(second), `"latest_step":2`) {
		t.Fatalf("second response should show second request count, got %s", second)
	}
}

func TestPersistentProcessTransportRestartsAfterUnexpectedExit(t *testing.T) {
	t.Parallel()

	transport := NewPersistentProcessTransport(
		os.Args[0],
		"-test.run=TestPersistentProcessTransportHelper",
		"--",
	)
	transport.SetEnv([]string{
		"GO_WANT_PERSISTENT_HELPER_PROCESS=1",
		"PERSISTENT_HELPER_MODE=exit_after_first",
	})
	defer func() { _ = transport.Close() }()

	first, err := transport.Call(context.Background(), []byte(`{"cmd":"ping-1"}`))
	if err != nil {
		t.Fatalf("first Call() error = %v", err)
	}
	second, err := transport.Call(context.Background(), []byte(`{"cmd":"ping-2"}`))
	if err != nil {
		t.Fatalf("second Call() error = %v", err)
	}

	pid1 := responseAgentID(t, first)
	pid2 := responseAgentID(t, second)
	if pid1 == pid2 {
		t.Fatalf("expected restarted helper process, got same pid marker %s", pid1)
	}
}

func TestPersistentProcessTransportCloseAllowsFreshRestart(t *testing.T) {
	t.Parallel()

	transport := NewPersistentProcessTransport(
		os.Args[0],
		"-test.run=TestPersistentProcessTransportHelper",
		"--",
	)
	transport.SetEnv([]string{
		"GO_WANT_PERSISTENT_HELPER_PROCESS=1",
		"PERSISTENT_HELPER_MODE=loop_success",
	})

	first, err := transport.Call(context.Background(), []byte(`{"cmd":"ping-1"}`))
	if err != nil {
		t.Fatalf("first Call() error = %v", err)
	}
	if err := transport.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	second, err := transport.Call(context.Background(), []byte(`{"cmd":"ping-2"}`))
	if err != nil {
		t.Fatalf("second Call() after Close error = %v", err)
	}
	defer func() { _ = transport.Close() }()

	pid1 := responseAgentID(t, first)
	pid2 := responseAgentID(t, second)
	if pid1 == pid2 {
		t.Fatalf("expected fresh process after Close(), got same pid marker %s", pid1)
	}
}

func TestPersistentProcessTransportContextCancelRestartsCleanly(t *testing.T) {
	t.Parallel()

	transport := NewPersistentProcessTransport(
		os.Args[0],
		"-test.run=TestPersistentProcessTransportHelper",
		"--",
	)
	transport.SetEnv([]string{
		"GO_WANT_PERSISTENT_HELPER_PROCESS=1",
		"PERSISTENT_HELPER_MODE=delay_first",
	})
	defer func() { _ = transport.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := transport.Call(ctx, []byte(`{"cmd":"slow-ping"}`))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Call() error = %v, want context deadline exceeded", err)
	}

	second, err := transport.Call(context.Background(), []byte(`{"cmd":"ping-2"}`))
	if err != nil {
		t.Fatalf("second Call() error = %v", err)
	}
	if !strings.Contains(string(second), `"latest_step":1`) {
		t.Fatalf("expected fresh process after cancelled call, got %s", second)
	}
}

func responseAgentID(t *testing.T, raw []byte) string {
	t.Helper()

	s := string(raw)
	marker := `"agent_id":"`
	idx := strings.Index(s, marker)
	if idx < 0 {
		t.Fatalf("agent_id marker missing in %s", s)
	}
	start := idx + len(marker)
	end := strings.Index(s[start:], `"`)
	if end < 0 {
		t.Fatalf("agent_id terminator missing in %s", s)
	}
	return s[start : start+end]
}

func TestPersistentProcessTransportHelper(t *testing.T) {
	t.Helper()

	if os.Getenv("GO_WANT_PERSISTENT_HELPER_PROCESS") != "1" {
		return
	}

	mode := os.Getenv("PERSISTENT_HELPER_MODE")
	requestCount := 0

	for {
		var line string
		if _, err := fmt.Fscanln(os.Stdin, &line); err != nil {
			os.Exit(0)
		}

		requestCount++
		if mode == "delay_first" && requestCount == 1 {
			time.Sleep(100 * time.Millisecond)
		}
		response := fmt.Sprintf(`{"kind":"state","state":{"workflow_id":"wf-persistent","agent_id":"pid-%d","status":"Running","latest_step":%d,"latest_root":"","events":[]}}`, os.Getpid(), requestCount)
		_, _ = fmt.Fprintln(os.Stdout, response)

		if mode == "exit_after_first" && requestCount == 1 {
			os.Exit(0)
		}
	}
}
