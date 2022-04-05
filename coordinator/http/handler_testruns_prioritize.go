package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *HttpServer) prioritizeTestRunHandler(
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
	tr.Priority = tr.Priority + 1
	h.tr.PersistTestRun(tr)
	writeJsonOK(w)
}
