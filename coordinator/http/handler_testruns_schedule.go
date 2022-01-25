package http

import (
	"encoding/json"
	"net/http"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (h *HttpServer) scheduleTestRunHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer r.Body.Close()
	var tr common.TestRun

	err := json.NewDecoder(r.Body).Decode(&tr)
	if err != nil {
		logging.Errorf("Error parsing request: %s", err.Error())
		http.Error(w, "Request format incorrect", 500)
		return
	}

	usr, err := h.UserFromRequest(r)
	if err != nil {
		logging.Errorf("Error determining user: %s", err.Error())
		http.Error(w, "Internal server error", 500)
		return
	}
	tr.CreatedByThumbprint = usr.Thumbprint

	sweepID, err := common.RandomID(12)
	if err != nil {
		logging.Errorf("Error getting randomness: %s", err.Error())
		http.Error(w, "Internal server error", 500)
		return
	}
	if tr.Repeat == 0 {
		tr.Repeat = 1
	}

	if tr.WatchtowerErrorCacheSize == 0 {
		tr.WatchtowerErrorCacheSize = 10000000
	}

	runs := common.ExpandSweepRun(&tr, sweepID)
	for i := range runs {
		h.tr.ScheduleTestRun(runs[i])
		if tr.SweepOneAtATime {
			break
		}
	}
	writeJsonOK(w)
}
