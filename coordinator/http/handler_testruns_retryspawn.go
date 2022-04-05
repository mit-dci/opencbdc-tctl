package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *HttpServer) retrySpawnHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	params := mux.Vars(r)
	runID := params["runID"]
	h.tr.RetrySpawn(runID)
	writeJsonOK(w)
}
