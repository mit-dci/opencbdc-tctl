package http

import (
	"net/http"

	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (srv *HttpServer) wsTokenHandler(w http.ResponseWriter, r *http.Request) {
	token, err := srv.wsTokenPayload(r)
	if err != nil {
		logging.Errorf("Error generating token payload: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
	writeJson(w, token)
}
