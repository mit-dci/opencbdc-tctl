package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *HttpServer) testRunResultsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	runID := params["runID"]

	run, ok := h.tr.GetTestRun(runID)
	if !ok {
		http.Error(w, "Not found", 404)
		return
	}

	results, err := h.tr.CalculateResults(run, false)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	writeJson(w, results)
}
