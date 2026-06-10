package launcher //nolint:testpackage // tests cover unexported release archive helpers

import "testing"

func TestContainerlabReleaseTarName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		goarch   string
		expected string
		wantErr  bool
	}{
		{
			name:     "amd64",
			version:  "0.76.0",
			goarch:   "amd64",
			expected: "containerlab_0.76.0_Linux_amd64.tar.gz",
		},
		{
			name:     "arm64",
			version:  "0.76.0",
			goarch:   "arm64",
			expected: "containerlab_0.76.0_Linux_arm64.tar.gz",
		},
		{
			name:    "unsupported",
			version: "0.76.0",
			goarch:  "386",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := containerlabReleaseTarName(tt.version, tt.goarch)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Fatalf("expected nil error, got %s", err)
			}

			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
