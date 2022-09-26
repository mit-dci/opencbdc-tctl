package http

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (h *HttpServer) continueSweep(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sweepID := params["sweepID"]
	trs := h.tr.GetTestRuns()

	for _, tr := range trs {
		if tr.SweepID == sweepID {
			h.tr.ContinueSweep(tr, sweepID)
			break
		}
	}
	writeJsonOK(w)
}
