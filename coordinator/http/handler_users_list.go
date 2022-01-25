package http

import "net/http"

func (srv *HttpServer) usersHandler(w http.ResponseWriter, r *http.Request) {
	writeJson(w, srv.users)
}
