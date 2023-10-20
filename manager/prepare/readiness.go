package prepare

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	clabernetesconstants "github.com/srl-labs/clabernetes/constants"
	clabernetesmanagertypes "github.com/srl-labs/clabernetes/manager/types"
)

const (
	readinessEndpoint = "/ready"
	readHeaderTimeout = 5 * time.Second
)

// endpoints initializes a simple http server to handle any required controller endpoints -- for now
// that just means a very simple readiness probe endpoint.
func endpoints(c clabernetesmanagertypes.Clabernetes) {
	handler := http.NewServeMux()

	handler.HandleFunc(readinessEndpoint, func(w http.ResponseWriter, _ *http.Request) {
		if c.IsReady() {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	server := http.Server{
		Addr:              fmt.Sprintf(":%d", clabernetesconstants.HealthProbePort),
		Handler:           handler,
		TLSNextProto:      make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	go func() {
		_ = server.ListenAndServe()
	}()
}
