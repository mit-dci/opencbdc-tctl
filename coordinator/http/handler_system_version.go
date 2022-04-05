package http

import (
	"net/http"
)

func (h *HttpServer) versionHandler(w http.ResponseWriter, r *http.Request) {
	writeJson(w, map[string]interface{}{"version": h.version})
}
