package http

import (
	"net/http"
)

func (srv *HttpServer) firstTimeAddUserHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if len(srv.users) > 0 {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	r.ParseMultipartForm(
		4 * 1024 * 1024,
	) // 4MB should suffice even the most ridiculously big certificates :)
	file, _, err := r.FormFile("firstTimeCert")
	if err != nil {
		http.Error(w, "File missing or invalid", http.StatusBadRequest)
		return
	}
	err = srv.addUser(w, r, file)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	srv.ReloadCerts()
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)

	_, err = w.Write([]byte(`
      <html>
         <head>
            <title>Authorize for CBDC-Test Controller</title>
         </head>
         <body>
		 	<h1>Success!</h1>
			 <p>Once you have imported the certificate you authorized in your browser, you can access the test controller <a href="` + srv.GetHttpsEndpoint("https", srv.httpsPort, r) + `">here</a></p>
		 </body>
      </html>`))

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

}
