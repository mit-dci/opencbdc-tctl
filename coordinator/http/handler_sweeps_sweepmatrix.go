package http

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (h *HttpServer) testRunSweepMatrixHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	sweepID := params["sweepID"]

	sweepIDs := []string{sweepID}
	if strings.Contains(sweepID, "|") {
		sweepIDs = strings.Split(sweepID, "|")
	}
	matrix, _, _ := h.tr.GenerateSweepMatrix(sweepIDs)

	writeJson(w, matrix)
}
