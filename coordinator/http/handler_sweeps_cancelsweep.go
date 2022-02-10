package http

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tct/common"
)

func (h *HttpServer) cancelSweepRuns(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sweepID := params["sweepID"]
	trs := h.tr.GetTestRuns()
	sweepRuns := make([]*common.TestRun, 0)
	for i := range trs {
		if trs[i].SweepID == sweepID &&
			(trs[i].Status == common.TestRunStatusQueued) {
			sweepRuns = append(sweepRuns, trs[i])
		}
	}
	for i := range sweepRuns {
		h.tr.UpdateStatus(
			sweepRuns[i],
			common.TestRunStatusCanceled,
			"Sweep canceled by user",
		)
	}
	writeJsonOK(w)
}
