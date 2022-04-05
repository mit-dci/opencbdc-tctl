package http

import (
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func (srv *HttpServer) deleteUserHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	vars := mux.Vars(r)
	fileToDelete := ""
	for _, u := range srv.users {
		if u.Thumbprint == vars["thumb"] {
			fileToDelete = u.certFile
		}
	}
	if fileToDelete == "" {
		http.Error(w, "Not found", 404)
		return
	}

	err := os.Remove(fileToDelete)
	if err != nil {
		http.Error(w, "Internal Server Error", 500)
		return
	}

	srv.ReloadCerts()

	writeJson(w, map[string]interface{}{"ok": true})
}
