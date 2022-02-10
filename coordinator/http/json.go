package http

import (
	"encoding/json"
	"net/http"

	"github.com/mit-dci/opencbdc-tct/logging"
)

func writeJson(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		logging.Errorf("Error writing JSON response: %v", err)
	}
}

func writeJsonOK(w http.ResponseWriter) {
	writeJson(w, map[string]interface{}{"ok": true})
}
