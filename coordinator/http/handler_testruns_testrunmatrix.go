package http

import (
	"net/http"
)

func (h *HttpServer) testRunMatrixHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	matrix, _, _ := h.tr.GenerateMatrix()
	writeJson(w, matrix)
}
