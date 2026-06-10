package image //nolint:testpackage // tests cover unexported native platform helper

import (
	"fmt"
	"runtime"
	"testing"
)

func TestNativePlatform(t *testing.T) {
	t.Parallel()

	expected := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if got := nativePlatform(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
