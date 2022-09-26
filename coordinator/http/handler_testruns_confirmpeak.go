package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

type confirmPeakBody struct {
	LowerBound float64 `json:"peakLB"`
	UpperBound float64 `json:"peakUB"`
}

func (h *HttpServer) testRunConfirmPeakHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer r.Body.Close()
	body := confirmPeakBody{}
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		logging.Errorf("Error parsing request: %s", err.Error())
		http.Error(w, "Request format incorrect", 500)
		return
	}

	params := mux.Vars(r)
	runID := params["runID"]

	run, ok := h.tr.GetTestRun(runID)
	if !ok {
		http.Error(w, "Not found", 404)
		return
	}

	run.Result.ThroughputPeakLB = body.LowerBound
	run.Result.ThroughputPeakUB = body.UpperBound
	h.tr.PersistTestRun(run)
	h.tr.ContinueSweep(run, run.SweepID)
	writeJsonOK(w)
}
