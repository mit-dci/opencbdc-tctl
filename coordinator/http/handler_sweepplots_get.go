package http

import (
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tctl/common"
)

func (h *HttpServer) getSavedSweepPlotHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	vars := mux.Vars(r)
	sweepID := vars["sweepID"]
	plotID := vars["plotID"]

	w.Header().Add("Content-Type", "image/png")
	w.Header().
		Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.png\"", plotID))
	http.ServeFile(
		w,
		r,
		filepath.Join(
			common.DataDir(),
			"testruns",
			"sweep-plots",
			sweepID,
			fmt.Sprintf("%s.png", plotID),
		),
	)
}
