package http

import (
	"net/http"
)

const (
	aliveRoute = "/alive"
)

func (m *manager) aliveHandler(w http.ResponseWriter, r *http.Request) {
	if !m.returnedReady {
		m.logRequest(r)
	}

	if m.managerReadyF() {
		w.WriteHeader(http.StatusOK)
	} else {
		m.logRequest(r)

		// reset so we log again, just dont wanna log this every 10s cuz its super annoying
		m.returnedReady = false

		w.WriteHeader(http.StatusInternalServerError)
	}
}
