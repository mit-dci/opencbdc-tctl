package http

import (
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

func (srv *HttpServer) addUserHandler(w http.ResponseWriter, r *http.Request) {
	err := srv.addUser(w, r, r.Body)
	writeJson(w, map[string]interface{}{"ok": err == nil})
	srv.ReloadCerts()
}

func (srv *HttpServer) addUser(
	w http.ResponseWriter,
	r *http.Request,
	fileReader io.ReadCloser,
) error {
	// Read body
	reqBody, err := ioutil.ReadAll(fileReader)
	if err != nil {
		logging.Warnf("Could not read body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	block, _ := pem.Decode([]byte(reqBody))

	if block == nil {
		logging.Warnf("Could not parse certificate from request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}

	newCert := make([]byte, 8)
	_, err = rand.Read(newCert)
	if err != nil {
		logging.Warnf("Could not read randomness: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
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
		return err
	}
	return fileReader.Close()
}
