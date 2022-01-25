package http

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (h *HttpServer) reconfigureMaxAgentsHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	params := mux.Vars(r)
	maxStr := params["max"]
	max, err := strconv.ParseInt(maxStr, 10, 32)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	err = h.tr.SetMaxAgents(int(max))
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	writeJsonOK(w)
}
