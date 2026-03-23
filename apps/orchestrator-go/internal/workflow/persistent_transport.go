package workflow

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type PersistentProcessTransport struct {
	binaryPath string
	args       []string
	env        []string

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *bytes.Buffer
}

func NewPersistentProcessTransport(binaryPath string, args ...string) *PersistentProcessTransport {
	return &PersistentProcessTransport{
		binaryPath: binaryPath,
		args:       args,
	}
}

func (t *PersistentProcessTransport) SetEnv(env []string) {
	t.env = append([]string(nil), env...)
}

func (t *PersistentProcessTransport) Call(ctx context.Context, requestJSON []byte) ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.callLocked(ctx, requestJSON, true)
}

func (t *PersistentProcessTransport) callLocked(ctx context.Context, requestJSON []byte, retry bool) ([]byte, error) {
	if err := t.ensureStartedLocked(); err != nil {
		return nil, err
	}

	resp, err := t.exchangeLocked(ctx, requestJSON)
	if err == nil {
		return resp, nil
	}

	_ = t.stopLocked()
	if isContextCancellation(err) {
		return nil, err
	}
	if !retry {
		return nil, err
	}

	if errStart := t.ensureStartedLocked(); errStart != nil {
		return nil, fmt.Errorf("restart runtime process: %w (original=%v)", errStart, err)
	}
	resp, retryErr := t.exchangeLocked(ctx, requestJSON)
	if retryErr != nil {
		_ = t.stopLocked()
		return nil, fmt.Errorf("persistent runtime retry failed: %w (original=%v)", retryErr, err)
	}
	return resp, nil
}

func (t *PersistentProcessTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stopLocked()
}

func (t *PersistentProcessTransport) ensureStartedLocked() error {
	if t.cmd != nil {
		return nil
	}
	if strings.TrimSpace(t.binaryPath) == "" {
		return fmt.Errorf("runtime binary path is empty")
	}

	cmd := exec.Command(t.binaryPath, t.args...)
	cmd.Env = append(os.Environ(), t.env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start runtime process: %w", err)
	}

	t.cmd = cmd
	t.stdin = stdin
	t.stdout = bufio.NewReader(stdoutPipe)
	t.stderr = stderr
	return nil
}

func (t *PersistentProcessTransport) exchangeLocked(ctx context.Context, requestJSON []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cmd := t.cmd
	stdin := t.stdin
	stdout := t.stdout
	stderr := t.stderr

	if _, err := io.WriteString(stdin, string(requestJSON)+"\n"); err != nil {
		return nil, fmt.Errorf("write runtime request: %w (stderr=%s)", err, strings.TrimSpace(stderr.String()))
	}

	type result struct {
		line []byte
		err  error
	}
	readCh := make(chan result, 1)
	go func() {
		line, err := stdout.ReadBytes('\n')
		readCh <- result{line: line, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-readCh:
		if res.err != nil && len(res.line) == 0 {
			if waitErr := cmd.Wait(); waitErr != nil {
				return nil, fmt.Errorf("read runtime response: %w (wait=%v, stderr=%s)", res.err, waitErr, strings.TrimSpace(stderr.String()))
			}
			return nil, fmt.Errorf("read runtime response: %w (stderr=%s)", res.err, strings.TrimSpace(stderr.String()))
		}

		resp := strings.TrimSpace(string(res.line))
		if resp == "" {
			return nil, fmt.Errorf("runtime response is empty (stderr=%s)", strings.TrimSpace(stderr.String()))
		}
		return []byte(resp), nil
	}
}

func (t *PersistentProcessTransport) stopLocked() error {
	if t.cmd == nil {
		return nil
	}

	cmd := t.cmd
	stdin := t.stdin

	t.cmd = nil
	t.stdin = nil
	t.stdout = nil
	t.stderr = nil

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if err := cmd.Wait(); err != nil {
		return nil
	}
	return nil
}

func isContextCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
