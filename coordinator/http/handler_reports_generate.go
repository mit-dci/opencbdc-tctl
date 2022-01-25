package http

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (h *HttpServer) generateReportHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	wg := sync.WaitGroup{}
	result := ""

	var def reportDefinition
	defer r.Body.Close()
	err := json.NewDecoder(r.Body).Decode(&def)
	if err != nil {
		logging.Warnf("Unable to create report: %v", err)
		http.Error(w, "Internal server error", 500)
		return
	}

	wg.Add(1)
	go func() {
		result = h.generateReport(def)
		wg.Done()
	}()
	wg.Wait()
	w.Header().Add("Content-Type", "text/html")
	w.Header().
		Add("Content-Disposition", "attachment; filename=\"report.html\"")
	_, err = w.Write([]byte(result))
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}

}
