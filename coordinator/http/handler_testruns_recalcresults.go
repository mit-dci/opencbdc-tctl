package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

type recalcBody struct {
	TrimZeroes    bool `json:"trimZeroes"`
	TrimZeroesEnd bool `json:"trimZeroesEnd"`
	TrimSamples   int  `json:"trimSamples"`
}

func (h *HttpServer) testRunRecalcResultsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer r.Body.Close()
	body := recalcBody{}
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

	run.TrimSamplesAtStart = body.TrimSamples
	run.TrimZeroesAtStart = body.TrimZeroes
	run.TrimZeroesAtEnd = body.TrimZeroesEnd
	h.tr.PersistTestRun(run)

	go func() {
		_, err := h.tr.CalculateResults(run, true)
		if err != nil {
			logging.Errorf("Error calculating results: %v", err)
		}
	}()
	writeJsonOK(w)
}
