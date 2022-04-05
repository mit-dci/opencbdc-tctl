package http

import "net/http"

func (h *HttpServer) sweepListHandler(w http.ResponseWriter, r *http.Request) {
	writeJson(w, h.listSweeps())
}
