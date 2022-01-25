package http

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (h *HttpServer) testRunLogHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	runID := params["runID"]

	run, ok := h.tr.GetTestRun(runID)
	if !ok {
		http.Error(w, "Not found", 404)
		return
	}

	w.Header().
		Add("Content-Disposition", fmt.Sprintf("attachment; filename=testrunlog_%s.txt", runID))
	w.Header().Add("Content-Type", "text/plain")
	_, err := w.Write([]byte(run.FullLog()))
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}
}
