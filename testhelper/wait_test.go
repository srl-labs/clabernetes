//nolint:testpackage // Exercise the polling deadlines and subprocess failures without a cluster.
package testhelper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWaitDeadline(t *testing.T) {
	t.Parallel()

	now := time.Now()

	for _, testCase := range []struct {
		name            string
		packageDeadline time.Time
		want            time.Time
	}{
		{"no package deadline", time.Time{}, now.Add(12 * time.Minute)},
		{"helper expires first", now.Add(30 * time.Minute), now.Add(12 * time.Minute)},
		{"reserve diagnostics and cleanup", now.Add(5 * time.Minute), now.Add(3 * time.Minute)},
		{"budget already consumed", now.Add(time.Minute), now.Add(-time.Minute)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := waitDeadline(now, testCase.packageDeadline, 12*time.Minute)
			if !got.Equal(testCase.want) {
				t.Fatalf("deadline = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestWaitForCommandOutputRetriesHungAttempt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	// The first attempt hangs after writing output. Killing it must allow the next attempt
	// to run, rather than consuming the whole polling budget.
	command := []string{"sh", "-c", `
if [ ! -f "$1" ]; then
    touch "$1"
    printf 'first attempt'
    exec sleep 60
fi
printf 'ready'
`, "sh", filepath.Join(t.TempDir(), "attempted")}
	output, err := waitForCommandOutput(
		ctx,
		command,
		"ready",
		100*time.Millisecond,
		time.Millisecond,
	)
	if err != nil || string(output) != "ready" {
		t.Fatalf("retry result = %q, %v", output, err)
	}
}

func TestWaitForCommandOutputDeadlinePreservesLastFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	// Matching output from a failed command must not satisfy the assertion. Cancellation
	// must also interrupt the long poll interval and preserve the last command result.
	output, err := waitForCommandOutput(
		ctx, []string{"sh", "-c", "printf ready; exit 7"}, "ready", time.Second, time.Hour,
	)
	if !errors.Is(err, context.DeadlineExceeded) ||
		!strings.Contains(err.Error(), "exit status 7") || string(output) != "ready" {
		t.Fatalf("timeout result = %q, %v", output, err)
	}
}

func TestWaitForCommandOutputParentBoundsAttempt(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	output, err := waitForCommandOutput(
		ctx, []string{"sh", "-c", "printf pending; exec sleep 60"}, "ready", time.Hour, time.Hour,
	)
	if !errors.Is(err, context.DeadlineExceeded) || string(output) != "pending" {
		t.Fatalf("parent timeout result = %q, %v", output, err)
	}
}

func TestDumpNamespaceDiagnosticsContinuesAfterFailure(t *testing.T) {
	fakeKubectl(t, `printf '%s\n' "$*"; exit 1`)

	var logs strings.Builder
	dumpNamespaceDiagnostics(t.Context(), "test-lab", func(format string, args ...any) {
		fmt.Fprintf(&logs, format, args...)
	})

	for _, want := range []string{
		"test-lab", "nodes.c9s.run,links.c9s.run", "services,endpointslices", "events",
		"--selector=c9s.run/direct-workload", "--all-containers=true", "--previous=true",
		"describe", "exit status 1",
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("diagnostics missing %q:\n%s", want, logs.String())
		}
	}
}

func TestDumpNamespaceDiagnosticsHasOverallDeadline(t *testing.T) {
	fakeKubectl(t, "exec sleep 60")

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	var logs strings.Builder
	dumpNamespaceDiagnostics(ctx, "test-lab", func(format string, args ...any) {
		fmt.Fprintf(&logs, format, args...)
	})
	if !strings.Contains(logs.String(), "diagnostics stopped: context deadline exceeded") {
		t.Fatalf("collection did not stop at its deadline:\n%s", logs.String())
	}
}

// This subprocess deliberately fails a wait near the package deadline. The parent verifies
// that the last output and workload diagnostics appear before cleanup, without Go's panic.
func TestKubectlWaitFailureBeforePackageTimeout(t *testing.T) {
	const childVariable = "C9S_TESTHELPER_WAIT_CHILD"
	if os.Getenv(childVariable) == "1" {
		defer t.Log("namespace cleanup")
		defer DumpNamespaceDiagnostics(t, "test-lab")

		KubectlWaitForOutput(t, 12*time.Minute, []string{"exec", "test-device"}, "0% packet loss")

		return
	}

	fakeKubectl(t, `
if [ "$1" = exec ]; then
    printf 'last ping output'
    exec sleep 60
fi
printf 'device diagnostics: %s\n' "$*"
`)
	t.Setenv(childVariable, "1")

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	// The production reserve is two minutes, leaving one second for this child's wait.
	cmd := exec.CommandContext(ctx, os.Args[0], //nolint:gosec // Re-execute this test binary.
		"-test.run=^TestKubectlWaitFailureBeforePackageTimeout$", "-test.timeout=121s",
	)
	output, err := cmd.CombinedOutput()
	if err == nil || ctx.Err() != nil {
		t.Fatalf("child should fail its assertion promptly: %v\n%s", err, output)
	}

	logs := string(output)
	for _, want := range []string{
		"context deadline exceeded", "last ping output", "device diagnostics:", "namespace cleanup",
	} {
		if !strings.Contains(logs, want) {
			t.Fatalf("failure output missing %q:\n%s", want, logs)
		}
	}
	if strings.Contains(logs, "panic:") ||
		strings.Index(logs, "device diagnostics:") > strings.Index(logs, "namespace cleanup") {
		t.Fatalf("diagnostics must precede cleanup and the package alarm:\n%s", logs)
	}
}

func fakeKubectl(t *testing.T, script string) {
	t.Helper()

	directory := t.TempDir()
	path := filepath.Join(directory, kubectl)
	//nolint:gosec // This temporary kubectl fixture must be executable.
	err := os.WriteFile(path, []byte("#!/bin/sh\n"+script+"\n"), 0o700)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH"))
}
