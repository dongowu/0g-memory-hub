package workflow

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type ProcessTransport struct {
	binaryPath string
	args       []string
	env        []string
}

func NewProcessTransport(binaryPath string, args ...string) *ProcessTransport {
	return &ProcessTransport{
		binaryPath: binaryPath,
		args:       args,
	}
}

func (t *ProcessTransport) SetEnv(env []string) {
	t.env = append([]string(nil), env...)
}

func (t *ProcessTransport) Call(ctx context.Context, requestJSON []byte) ([]byte, error) {
	if strings.TrimSpace(t.binaryPath) == "" {
		return nil, fmt.Errorf("runtime binary path is empty")
	}

	cmd := exec.CommandContext(ctx, t.binaryPath, t.args...)
	cmd.Env = append(os.Environ(), t.env...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start runtime process: %w", err)
	}

	_, writeErr := io.WriteString(stdin, string(requestJSON)+"\n")
	closeErr := stdin.Close()
	if writeErr != nil {
		_ = cmd.Wait()
		return nil, fmt.Errorf("write runtime request: %w", writeErr)
	}
	if closeErr != nil {
		_ = cmd.Wait()
		return nil, fmt.Errorf("close runtime stdin: %w", closeErr)
	}

	reader := bufio.NewReader(stdout)
	line, readErr := reader.ReadBytes('\n')
	if readErr != nil && len(line) == 0 {
		waitErr := cmd.Wait()
		if ctx.Err() != nil {
			return nil, fmt.Errorf("runtime call canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("read runtime response: %w (wait=%v, stderr=%s)", readErr, waitErr, strings.TrimSpace(stderr.String()))
	}

	waitErr := cmd.Wait()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("runtime call canceled: %w", ctx.Err())
	}
	if waitErr != nil {
		return nil, fmt.Errorf("runtime process failed: %w (stderr=%s)", waitErr, strings.TrimSpace(stderr.String()))
	}

	resp := strings.TrimSpace(string(line))
	if resp == "" {
		return nil, fmt.Errorf("runtime response is empty (stderr=%s)", strings.TrimSpace(stderr.String()))
	}
	return []byte(resp), nil
}
