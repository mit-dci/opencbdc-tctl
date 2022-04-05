package http

import (
	"net/http"
)

func (h *HttpServer) systemMaintenanceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method == "GET" {
		writeJson(
			w,
			map[string]interface{}{"maintenance": h.coord.GetMaintenance()},
		)
		return
	}
	if r.Method == "PUT" {
		h.coord.SetMaintenance(!h.coord.GetMaintenance())
		writeJsonOK(w)
		return
	}
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
