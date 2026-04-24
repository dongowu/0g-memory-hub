package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dongowu/0g-memory-hub/apps/orchestrator-go/internal/workflow"
)

func TestWorkflowCommandIncludesVerifyCommand(t *testing.T) {
	t.Parallel()

	foundWorkflow := false
	foundVerify := false

	for _, c := range rootCmd.Commands() {
		if c.Name() != "workflow" {
			continue
		}
		foundWorkflow = true
		for _, sub := range c.Commands() {
			if sub.Name() == "verify" {
				foundVerify = true
				break
			}
		}
		break
	}

	if !foundWorkflow {
		t.Fatal("workflow command not registered on root command")
	}
	if !foundVerify {
		t.Fatal("verify command not registered on workflow command")
	}
}

func TestWireWorkflowMVPDepsFormatsLocalStorageWarnings(t *testing.T) {
	tempDir := t.TempDir()
	blockingPath := filepath.Join(tempDir, "not-a-directory")
	if err := os.WriteFile(blockingPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocking path: %v", err)
	}

	store, err := workflow.NewFileStore(filepath.Join(tempDir, "workflows.json"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	svc := workflow.NewService(store)

	t.Setenv("ORCH_DATA_DIR", blockingPath)
	t.Setenv("ORCH_CHAIN_PRIVATE_KEY", "")

	stderr := captureStderr(t, func() {
		closer := wireWorkflowMVPDeps(svc)
		if closer == nil {
			t.Fatal("wireWorkflowMVPDeps() returned nil closer")
		}
		if err := closer.Close(); err != nil {
			t.Fatalf("closer.Close() error = %v", err)
		}
	})

	if !strings.Contains(stderr, "warning: could not create local storage:") {
		t.Fatalf("stderr missing local storage warning: %q", stderr)
	}
	if strings.Contains(stderr, "%!") {
		t.Fatalf("stderr contains fmt formatting artifact: %q", stderr)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}

	os.Stderr = writer
	defer func() {
		os.Stderr = original
	}()

	outputCh := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, reader)
		outputCh <- buf.String()
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	output := <-outputCh
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() error = %v", err)
	}

	return output
}
