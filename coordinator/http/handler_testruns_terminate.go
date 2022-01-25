package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *HttpServer) terminateTestRunHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	defer r.Body.Close()
	params := mux.Vars(r)
	runID := params["runID"]
	h.tr.Terminate(runID)
	writeJsonOK(w)
}
