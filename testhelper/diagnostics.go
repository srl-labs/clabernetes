package testhelper

import (
	"context"
	"testing"
	"time"
)

const diagnosticCommandTimeout = 10 * time.Second

// DumpNamespaceDiagnostics records workload state and device/connectivity logs before cleanup.
// Collection is best effort and bounded even when the API server or a log request is stalled.
// Only Node/Link status is printed; their specs can contain startup configuration and credentials.
func DumpNamespaceDiagnostics(t *testing.T, namespace string) {
	t.Helper()

	deadline := time.Now().Add(diagnosticTimeout)
	if packageDeadline, ok := t.Deadline(); ok {
		latest := packageDeadline.Add(-cleanupReserve)
		if latest.Before(deadline) {
			deadline = latest
		}
	}

	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()

	dumpNamespaceDiagnostics(ctx, namespace, t.Logf)
}

func dumpNamespaceDiagnostics(
	ctx context.Context,
	namespace string,
	logf func(string, ...any),
) {
	commands := [][]string{
		{string(Get), "pods", "-o", "wide"},
		{
			string(Get), "nodes.c9s.run,links.c9s.run", "-o",
			`jsonpath={range .items[*]}{.kind}/{.metadata.name}{"\n"}{.status}{"\n"}{end}`,
		},
		{string(Get), "services,endpointslices", "-o", "yaml"},
		{string(Get), "events", "--sort-by=.lastTimestamp"},
		{
			"logs", "--selector=c9s.run/direct-workload", "--all-containers=true",
			"--prefix=true", "--timestamps=true", "--tail=200", "--ignore-errors=true",
		},
		{
			"logs", "--selector=c9s.run/direct-workload", "--all-containers=true",
			"--prefix=true", "--timestamps=true", "--tail=200", "--ignore-errors=true",
			"--previous=true",
		},
		{"describe", "pods"},
	}

	for _, arguments := range commands {
		if ctx.Err() != nil {
			logf("namespace %q diagnostics stopped: %v", namespace, ctx.Err())

			return
		}

		command := append([]string{kubectl, "--namespace", namespace}, arguments...)
		output, err := boundedCommandOutput(ctx, diagnosticCommandTimeout, command)
		logf("diagnostic %q (error: %v):\n%s", command, err, output)
	}
}
