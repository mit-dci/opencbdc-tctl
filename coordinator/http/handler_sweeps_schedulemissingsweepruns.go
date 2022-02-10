package http

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
)

func (h *HttpServer) scheduleMissingSweepRuns(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	sweepID := params["sweepID"]
	trs := h.tr.GetTestRuns()

	expectedRuns := common.FindMissingSweepRuns(trs, sweepID)

	logging.Infof(
		"Sweep ID is missing %d runs, scheduling them...",
		len(expectedRuns),
	)
	for i := range expectedRuns {
		for j := range expectedRuns[i].Roles {
			expectedRuns[i].Roles[j].AgentID = -1
		}
		expectedRuns[i].AWSInstancesStopped = false
		h.tr.ScheduleTestRun(expectedRuns[i])
	}
	writeJsonOK(w)
}
