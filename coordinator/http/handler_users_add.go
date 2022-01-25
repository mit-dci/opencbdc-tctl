package http

import (
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

func (srv *HttpServer) addUserHandler(w http.ResponseWriter, r *http.Request) {
	// Read body
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logging.Warnf("Could not read body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	block, _ := pem.Decode([]byte(reqBody))

	if block == nil {
		logging.Warnf("Could not parse certificate from request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newCert := make([]byte, 8)
	_, err = rand.Read(newCert)
	if err != nil {
		logging.Warnf("Could not read randomness: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = ioutil.WriteFile(
		filepath.Join(
			common.DataDir(),
			fmt.Sprintf("certs/users/%x.crt", newCert),
		),
		reqBody,
		0600,
	)
	if err != nil {
		logging.Warnf("Could not write user certificate file: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	r.Body.Close()

	writeJson(w, map[string]interface{}{"ok": true})
	srv.ReloadCerts()
}
