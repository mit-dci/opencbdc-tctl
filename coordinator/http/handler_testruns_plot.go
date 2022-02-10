package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
)

func (h *HttpServer) testRunPlotHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	runID := params["runID"]
	plot := params["plot"]
	tr, ok := h.tr.GetTestRun(runID)
	if !ok {
		http.Error(w, "Not found", 404)
		return
	}
	cmdID := ""
	plotType := ""
	extension := "png"
	if strings.HasPrefix(plot, "perf_") {

		// Generate it!
		cmdIDOffset := strings.LastIndex(plot, "_")
		cmdID = plot[cmdIDOffset+1:]

		versionOffset := strings.LastIndex(plot[:cmdIDOffset], "_")
		plotTypeOffset := strings.Index(plot[:versionOffset], "_")
		plotType = plot[plotTypeOffset+1 : versionOffset]
		if plotType == "flamegraph" {
			extension = "svg"
		}
	}

	path := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/plots/%s.%s", runID, plot, extension),
	)
	if s, err := os.Stat(path); os.IsNotExist(err) ||
		(plot == "flamegraph" && s.Size() < 1000) {
		if strings.HasPrefix(
			plot,
			"perf_",
		) { // Performance plots can be generated on-the-fly
			err = h.tr.CalculatePerformancePlot(tr, cmdID, plotType)
			if err != nil {
				logging.Errorf("Error calculating performance plot: %v", err)
				// If script wrote to the file a bit already (corrupted), ensure
				// it's gone
				// so we retry next time
				os.Remove(path)
				http.Error(w, "Internal Server Error", 500)
				return
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				http.Error(w, "Not found", 404)
				return
			}
		} else {
			http.Error(w, "Not found", 404)
			return
		}
	}

	f, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}
	defer f.Close()
	if extension == "png" {
		w.Header().Add("Content-Type", "image/png")
	} else {
		w.Header().Add("Content-Type", "image/svg+xml")
	}

	w.WriteHeader(200)

	_, err = io.Copy(w, f)
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}
}
