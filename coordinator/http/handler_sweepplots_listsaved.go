package http

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (h *HttpServer) listSavedSweepPlotsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	vars := mux.Vars(r)
	plots := make([]savedSweepPlot, 0)
	sweepID := vars["sweepID"]
	sweepPlotsDir := filepath.Join(
		common.DataDir(),
		"testruns",
		"sweep-plots",
		sweepID,
	)
	err := filepath.Walk(
		sweepPlotsDir,
		func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(path, ".json") {
				lastSlash := strings.LastIndex(path, "/")
				plotID := strings.ReplaceAll(path[lastSlash+1:], ".json", "")
				raw := map[string]interface{}{}
				f, err := os.OpenFile(path, os.O_RDONLY, 0644)
				if err != nil {
					logging.Warnf("Unable to load saved plot %s: %v", path, err)
					return nil
				}
				defer f.Close()
				s, err := os.Stat(path)
				if err != nil {
					logging.Warnf("Unable to stat saved plot %s: %v", path, err)
					return nil
				}

				err = json.NewDecoder(f).Decode(&raw)
				if err != nil {
					logging.Warnf("Unable to read saved plot %s: %v", path, err)
					return nil
				}
				title := ""
				cfg := map[string]interface{}{}
				cfgRaw, ok := raw["request"]
				if ok {
					cfg, ok = cfgRaw.(map[string]interface{})
					if ok {
						titleRaw, ok := cfg["title"]
						if ok {
							title, _ = titleRaw.(string)
						}
					}
				}
				plots = append(plots, savedSweepPlot{
					ID:      plotID,
					Title:   title,
					Date:    s.ModTime(),
					Request: cfg,
				})
			}
			return nil
		},
	)
	if err != nil {
		logging.Errorf("Error reading saved sweepplots: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJson(w, plots)
}
