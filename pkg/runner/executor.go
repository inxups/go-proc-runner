package runner

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"time"
)

type ExecResult struct {
	Stdout   string
	Stderr   string
	Duration time.Duration
	TimedOut bool
	Err      error
}

func RunWithTimeOut(timeout time.Duration, name string, args ...string) ExecResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	return ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded),
		Err:      err,
	}
}
