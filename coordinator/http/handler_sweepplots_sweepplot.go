package http

import (
	"fmt"
	"net/http"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (h *HttpServer) sweepPlotHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	plotOutput, err := h.generateSweepPlot(r.Body)
	if err != nil {
		logging.Warnf("Unable to create plot: %v", err)
		http.Error(w, "Internal server error", 500)
		return
	}
	randomID, err := common.RandomID(12)
	if err != nil {
		logging.Warnf("Unable to create plot: %v", err)
		http.Error(w, "Internal server error", 500)
		return
	}

	randName := fmt.Sprintf("plot-%s", randomID)
	w.Header().Add("Content-Type", "image/png")
	w.Header().
		Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.png\"", randName))
	_, err = w.Write(plotOutput)
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}
}
