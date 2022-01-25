package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *HttpServer) testRunDetailsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	runID := params["runID"]
	tr, ok := h.tr.GetTestRun(runID)
	if !ok {
		http.Error(w, "Not found", 404)
		return
	}
	writeJson(w, tr)
}
