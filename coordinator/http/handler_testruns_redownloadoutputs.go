package http

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tct/coordinator"
	"github.com/mit-dci/opencbdc-tct/logging"
)

func (h *HttpServer) redownloadOutputsHandler(
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

	go func() {
		err := h.tr.RedownloadTestOutputsFromS3(tr)
		success := true
		errorString := ""
		if err != nil {
			logging.Errorf("Failed redownloading outputs from S3: %v", err)
			success = false
			errorString = err.Error()
		}

		h.events <- coordinator.Event{
			Type: coordinator.EventTypeRedownloadComplete,
			Payload: coordinator.RedownloadCompletePayload{
				Success:   success,
				TestRunID: tr.ID,
				Error:     errorString,
			},
		}
	}()
	writeJsonOK(w)
}
