package http

import (
	"net/http"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

func (h *HttpServer) initialStateHandler(
	w http.ResponseWriter,
	r *http.Request,
) {

	commits, err := h.src.GetGitLog(0, 50, true)
	if err != nil {
		logging.Errorf("Error getting git log: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	usr, err := h.UserFromRequest(r)
	if err != nil {
		logging.Errorf("Error getting user from request: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	token, err := h.wsTokenPayload(r)
	if err != nil {
		logging.Errorf("Error getting websocket token: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	writeJson(w, map[string]interface{}{
		"commits":         commits,
		"agentCount":      h.coord.GetAgentCount(),
		"launchTemplates": h.awsm.LaunchTemplates(),
		"architectures":   common.AvailableArchitectures,
		"version":         h.version,
		"maintenance":     h.coord.GetMaintenance(),
		"config":          h.tr.Config(),
		"testruns":        h.frontendTestRunList(),
		"me":              usr,
		"users":           h.users,
		"sweeps":          h.listSweeps(),
		"websocket":       token,
		"onlineUsers":     len(websockets),
		"sweepPlotConfig": h.getSweepPlotConfig(),
		"testRunFields":   h.testRunFieldList(),
		"perfGraphs":      h.performancePlotTypes(),
		"shardSeeds":      h.awsm.GetAvailableSeeds(),
	})
}
