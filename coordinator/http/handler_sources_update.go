package http

import "net/http"

func (h *HttpServer) sourcesUpdateHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	err := h.src.EnsureSourcesUpdated()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJsonOK(w)
}
