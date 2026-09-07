package testhelper

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const (
	kubectlAttemptTimeout = 30 * time.Second
	kubectlPollInterval   = 5 * time.Second
	commandWaitDelay      = time.Second
	diagnosticTimeout     = time.Minute
	cleanupReserve        = time.Minute
)

// KubectlWaitForOutput retries a bounded kubectl command until it succeeds and contains expect.
// It stops before the package deadline, reserving time for diagnostics and namespace cleanup.
func KubectlWaitForOutput(
	t *testing.T,
	timeout time.Duration,
	arguments []string,
	expect string,
) {
	t.Helper()

	packageDeadline, _ := t.Deadline()
	ctx, cancel := context.WithDeadline(
		t.Context(), waitDeadline(time.Now(), packageDeadline, timeout),
	)
	defer cancel()

	command := append([]string{kubectl}, arguments...)
	output, err := waitForCommandOutput(
		ctx, command, expect, kubectlAttemptTimeout, kubectlPollInterval,
	)
	if err != nil {
		t.Fatalf("command %q: %v\nlast output:\n%s", command, err, output)
	}
}

func waitDeadline(now, packageDeadline time.Time, timeout time.Duration) time.Time {
	deadline := now.Add(timeout)
	if !packageDeadline.IsZero() {
		latest := packageDeadline.Add(-diagnosticTimeout - cleanupReserve)
		if latest.Before(deadline) {
			deadline = latest
		}
	}

	return deadline
}

func waitForCommandOutput(
	ctx context.Context,
	command []string,
	expect string,
	attemptTimeout, pollInterval time.Duration,
) ([]byte, error) {
	var output []byte
	var lastErr error

	for ctx.Err() == nil {
		output, lastErr = boundedCommandOutput(ctx, attemptTimeout, command)
		if lastErr == nil && strings.Contains(string(output), expect) {
			return output, nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}

	err := fmt.Errorf("waiting for %q: %w", expect, ctx.Err())
	if lastErr != nil {
		err = fmt.Errorf("%w; last command error: %w", err, lastErr)
	}

	return output, err
}

func boundedCommandOutput(
	ctx context.Context,
	timeout time.Duration,
	command []string,
) ([]byte, error) {
	attemptCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext( //nolint:gosec // Test-controlled commands.
		attemptCtx,
		command[0],
		command[1:]...,
	)
	// An exec client or wrapper may leave descendants holding its output pipes open after it
	// exits. Bound pipe draining as well as the process itself.
	cmd.WaitDelay = commandWaitDelay

	output, err := cmd.CombinedOutput()
	if attemptCtx.Err() != nil {
		return output, attemptCtx.Err()
	}

	return output, err
}
