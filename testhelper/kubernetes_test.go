package testhelper

import (
	"strings"
	"testing"
)

func TestNormalizeDeployment(t *testing.T) {
	normalized := NormalizeDeployment(t, []byte(`apiVersion: apps/v1
kind: Deployment
spec:
  template:
    metadata:
      labels:
        clabernetes/name: topology-basic-srl1
    spec:
      containers:
        - name: srl1
          image: ghcr.io/nokia/srlinux
status:
  terminatingReplicas: 0
`))

	if got := strings.TrimSpace(string(YQCommand(t, normalized, `.spec.template.spec.containers[0] | has("image")`))); got != "false" {
		t.Fatalf("expected deployment image to be removed, got %q", got)
	}

	if got := strings.TrimSpace(string(YQCommand(t, normalized, ".spec.template.metadata.creationTimestamp"))); got != "null" {
		t.Fatalf("expected pod template creationTimestamp to normalize to null, got %q", got)
	}

	if got := strings.TrimSpace(string(YQCommand(t, normalized, ".status | length"))); got != "0" {
		t.Fatalf("expected deployment status to normalize to empty object, got %q", got)
	}
}
