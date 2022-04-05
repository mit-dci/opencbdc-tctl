package http

import (
	"net/http"
)

func (h *HttpServer) testRunMatrixCsvHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	mtrx, _, _ := h.tr.GenerateMatrix()
	h.matrixToCsv(mtrx, w, r, "result-matrix.csv")
}
