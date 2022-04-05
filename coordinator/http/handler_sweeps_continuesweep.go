package http

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

func (h *HttpServer) continueSweep(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sweepID := params["sweepID"]
	trs := h.tr.GetTestRuns()

	expectedRuns := common.FindMissingSweepRuns(trs, sweepID)

	logging.Infof(
		"Sweep ID is missing %d runs, scheduling the first...",
		len(expectedRuns),
	)
	for j := range expectedRuns[0].Roles {
		expectedRuns[0].Roles[j].AgentID = -1
	}
	expectedRuns[0].AWSInstancesStopped = false
	h.tr.ScheduleTestRun(expectedRuns[0])
	writeJsonOK(w)
}
