package http

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

func (h *HttpServer) testRunOutputsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	runID := params["runID"]

	_, ok := h.tr.GetTestRun(runID)
	if !ok {
		http.Error(w, "Not found", 404)
		return
	}

	path := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/outputs", runID),
	)
	archivePath := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/outputs.tar.gz", runID),
	)
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		f, err := os.OpenFile(archivePath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			logging.Errorf("Error creating test run archive: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		err = common.CreateArchiveToStream(path, f)
		f.Close()
		if err != nil {
			logging.Errorf("Error creating test run archive: %v", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}
	}
	w.Header().Add("Content-Type", "application/tar+gzip")
	w.Header().
		Add("Content-Disposition", fmt.Sprintf("attachment; filename=\"testrun-output-%s.tar.gz\"", runID))
	http.ServeFile(w, r, archivePath)
}
