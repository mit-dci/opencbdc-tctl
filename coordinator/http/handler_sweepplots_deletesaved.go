package http

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tctl/common"
)

func (h *HttpServer) deleteSavedSweepPlotHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	vars := mux.Vars(r)
	sweepID := vars["sweepID"]
	plotID := vars["plotID"]

	os.Remove(
		filepath.Join(
			common.DataDir(),
			"testruns",
			"sweep-plots",
			sweepID,
			fmt.Sprintf("%s.png", plotID),
		),
	)
	os.Remove(
		filepath.Join(
			common.DataDir(),
			"testruns",
			"sweep-plots",
			sweepID,
			fmt.Sprintf("%s.json", plotID),
		),
	)

	writeJsonOK(w)
}
