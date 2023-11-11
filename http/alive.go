package http

import (
	"net/http"
)

const (
	aliveRoute = "/alive"
)

func (m *manager) aliveHandler(w http.ResponseWriter, r *http.Request) {
	m.logRequest(r)

	if m.managerReadyF() {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
