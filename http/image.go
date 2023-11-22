package http

import "net/http"

const (
	imageRoute = "/image"
)

func (m *manager) imageHandler(w http.ResponseWriter, r *http.Request) {
	m.logRequest(r)

	w.WriteHeader(http.StatusInternalServerError)
}
