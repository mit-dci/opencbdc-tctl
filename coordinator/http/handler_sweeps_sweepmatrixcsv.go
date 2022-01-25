package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func (h *HttpServer) testRunSweepMatrixCsvHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	sweepID := params["sweepID"]
	sweepIDs := []string{sweepID}
	if strings.Contains(sweepID, "|") {
		sweepIDs = strings.Split(sweepID, "|")
	}
	mtrx, _, _ := h.tr.GenerateSweepMatrix(sweepIDs)
	h.matrixToCsv(
		mtrx,
		w,
		r,
		fmt.Sprintf("result-matrix-sweeps-%s.csv", strings.Join(sweepIDs, "-")),
	)
}
