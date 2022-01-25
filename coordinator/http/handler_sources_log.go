package http

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (h *HttpServer) sourcesLogHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	limit, _ := strconv.Atoi(params["limit"])
	if limit == 0 {
		limit = 50
	}
	page, _ := strconv.Atoi(params["page"])

	logs, err := h.src.GetGitLog(limit*page, limit*page+limit)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJson(w, logs)
}
