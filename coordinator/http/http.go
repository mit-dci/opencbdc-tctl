package http

import (
	"crypto/md5"
	"crypto/tls"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/coordinator"
	"github.com/mit-dci/opencbdc-tct/coordinator/agents"
	"github.com/mit-dci/opencbdc-tct/coordinator/awsmgr"
	"github.com/mit-dci/opencbdc-tct/coordinator/sources"
	"github.com/mit-dci/opencbdc-tct/coordinator/testruns"
	"github.com/mit-dci/opencbdc-tct/logging"
	"github.com/rs/cors"
)

type HttpServer struct {
	srv                        *http.Server
	src                        *sources.SourcesManager
	am                         *agents.AgentsManager
	awsm                       *awsmgr.AwsManager
	tr                         *testruns.TestRunManager
	coord                      *coordinator.Coordinator
	events                     chan coordinator.Event
	httpsWithoutClientCertPort int
	httpsPort                  int
	users                      []*SystemUser
	roots                      *x509.CertPool
	certificate                tls.Certificate
	wsTokens                   sync.Map
	version                    string
}

type SystemUser struct {
	CN           string `json:"name"`
	Email        string `json:"email"`
	Organization string `json:"org"`
	Thumbprint   string `json:"thumbPrint"`
	certFile     string
}

func NewHttpServer(
	c *coordinator.Coordinator,
	s *sources.SourcesManager,
	a *agents.AgentsManager,
	t *testruns.TestRunManager,
	ev chan coordinator.Event,
	awsm *awsmgr.AwsManager,
	port int,
	version string,
) (*HttpServer, error) {
	httpSrv := HttpServer{
		coord:    c,
		src:      s,
		am:       a,
		tr:       t,
		events:   ev,
		users:    []*SystemUser{},
		wsTokens: sync.Map{},
		awsm:     awsm,
		version:  version,
	}
	httpSrv.httpsWithoutClientCertPort, _ = strconv.Atoi(
		os.Getenv("HTTPS_WITHOUT_CLIENT_CERT_PORT"),
	)
	if httpSrv.httpsWithoutClientCertPort == 0 {
		httpSrv.httpsWithoutClientCertPort = 444
	}

	httpSrv.httpsPort, _ = strconv.Atoi(os.Getenv("HTTPS_PORT"))
	if httpSrv.httpsPort == 0 {
		httpSrv.httpsPort = 443
	}

	r := mux.NewRouter()

	// Websocket token
	r.HandleFunc("/api/wsToken", httpSrv.wsTokenHandler).Methods("GET")

	// Initial State
	r.HandleFunc("/api/initialState", httpSrv.initialStateHandler).
		Methods("GET")

	// Users
	r.HandleFunc("/api/users", httpSrv.usersHandler).Methods("GET")
	r.HandleFunc("/api/users", httpSrv.addUserHandler).Methods("POST")
	r.HandleFunc("/api/users/{thumb}", httpSrv.deleteUserHandler).
		Methods("DELETE")

	// Maintenance mode
	r.HandleFunc("/api/maintenance", NoCache(httpSrv.systemMaintenanceHandler)).
		Methods("GET", "PUT")

	// Version
	r.HandleFunc("/api/version", httpSrv.versionHandler).Methods("GET")

	// Test runs
	r.HandleFunc("/api/testruns/sweeps", NoCache(httpSrv.sweepListHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/matrix", NoCache(httpSrv.testRunMatrixHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/sweepMatrix/{sweepID}", NoCache(httpSrv.testRunSweepMatrixHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/sweepMatrixCsv/{sweepID}", NoCache(httpSrv.testRunSweepMatrixCsvHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/matrixcsv", NoCache(httpSrv.testRunMatrixCsvHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/maxagents/{max}", NoCache(httpSrv.reconfigureMaxAgentsHandler)).
		Methods("PUT")
	r.HandleFunc("/api/testruns/schedule", httpSrv.scheduleTestRunHandler).
		Methods("POST")
	r.HandleFunc("/api/testruns/{runID}/prioritize", httpSrv.prioritizeTestRunHandler).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/redownloadOutputs", httpSrv.redownloadOutputsHandler).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/log/{offset}", NoCache(httpSrv.testRunLogHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/log", NoCache(httpSrv.testRunLogHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/details", NoCache(httpSrv.testRunDetailsHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/results", httpSrv.testRunResultsHandler).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/results/recalc", httpSrv.testRunRecalcResultsHandler).
		Methods("POST")
	r.HandleFunc("/api/testruns/{runID}/plot/{plot}", NoCache(httpSrv.testRunPlotHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/outputs", NoCache(httpSrv.testRunOutputsHandler)).
		Methods("GET")
	r.HandleFunc("/api/testruns/{runID}/terminate", httpSrv.terminateTestRunHandler).
		Methods("PUT")
	r.HandleFunc("/api/testruns/{runID}/retrySpawn", httpSrv.retrySpawnHandler).
		Methods("PUT")
	r.HandleFunc("/api/testruns/{runID}/bandwidth", httpSrv.testRunBandwidthData).
		Methods("GET")

	// Sweeps
	r.HandleFunc("/api/sweeps/{sweepID}/fixMissing", httpSrv.scheduleMissingSweepRuns).
		Methods("GET")
	r.HandleFunc("/api/sweeps/{sweepID}/continue", httpSrv.continueSweep).
		Methods("GET")
	r.HandleFunc("/api/sweeps/{sweepID}/cancel", httpSrv.cancelSweepRuns).
		Methods("GET")

	// Commands
	r.HandleFunc("/api/commands/{cmdID}/output/{stream}", httpSrv.commandOutputHandler).
		Methods("GET")
	r.HandleFunc("/api/commands/{cmdID}/output/{stream}/{full}", httpSrv.commandOutputHandler).
		Methods("GET")

	// Report
	r.HandleFunc("/api/generateReport", NoCache(httpSrv.generateReportHandler)).
		Methods("POST")

	// Sweep plots
	r.HandleFunc("/api/sweepplot", NoCache(httpSrv.sweepPlotHandler)).
		Methods("POST")
	r.HandleFunc("/api/sweepplot/saved/{sweepID}", NoCache(httpSrv.listSavedSweepPlotsHandler)).
		Methods("GET")
	r.HandleFunc("/api/sweepplot/saved/{sweepID}/{plotID}", NoCache(httpSrv.getSavedSweepPlotHandler)).
		Methods("GET")
	r.HandleFunc("/api/sweepplot/saved/{sweepID}/{plotID}", httpSrv.deleteSavedSweepPlotHandler).
		Methods("DELETE")

	// Sources
	r.HandleFunc("/api/sources/log", httpSrv.sourcesLogHandler).Methods("GET")
	r.HandleFunc("/api/sources/update", httpSrv.sourcesUpdateHandler).
		Methods("POST")

	spa := spaHandler{staticPath: "frontend", indexPath: "index.html"}
	r.PathPrefix("/").Handler(spa)

	// TODO: Probably tighten this once we take this in production
	var corsOpt = cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"}, // DEBUG
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowCredentials: true,
	})

	httpSrv.srv = &http.Server{
		Handler: handlers.CompressHandler(corsOpt.Handler(r)),
		Addr:    fmt.Sprintf(":%d", httpSrv.httpsPort),
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		TLSConfig: &tls.Config{
			ClientAuth:         tls.RequireAndVerifyClientCert,
			GetConfigForClient: httpSrv.GetConfigForClient,
		},
	}

	httpSrv.ReloadCerts()
	return &httpSrv, nil
}

func (srv *HttpServer) HttpsRedirect(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	target := srv.GetHttpsEndpoint("https", srv.httpsPort, req)
	target += req.URL.Path

	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	http.Redirect(w, req, target,
		// see comments below and consider the codes 308, 302, or 301
		http.StatusTemporaryRedirect)
}

func (srv *HttpServer) AuthorizeHandler(
	w http.ResponseWriter,
	req *http.Request,
) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)

	postForm := ""
	if len(srv.users) == 0 {
		postForm = `<h2>Authorize first user</h2><p>Since your system does not have any configured users yet, you can add the first one from this unauthenticated endpoint. Browse to the .crt file you created to add it to the list of authenticated users.</p>
		<form enctype="multipart/form-data" action="/firstTimeAuth" method="post">
		<input type="file" name="firstTimeCert" />
		<button type="submit">Authorize</button>
		</form>`
	}

	_, err := w.Write([]byte(`
      <html>
         <head>
            <title>Authorize for CBDC-Test Controller</title>
            <script language="javascript">
            <!--
               function updateScript() {
                  document.getElementById('script').innerText = "curl https://` + req.Host + `/create-cert.sh | " +
                     "CERT_CN=\"" + document.getElementById('fullName').value + "\" " +
                     "CERT_COUNTRY=\"" + document.getElementById('country').value + "\"  " +
                     "CERT_EMAIL=\"" + document.getElementById('email').value + "\" " +
                     "CERT_STATE=\"" + document.getElementById('state').value + "\" " +
                     "CERT_LOCALITY=\"" + document.getElementById('locality').value + "\" " +
                     "CERT_ORG=\"" + document.getElementById('company').value + "\" " +
                     "CERT_OU=\"" + document.getElementById('ou').value + "\" " +
                     "bash";
               }

               function copyScript() {
                  document.getElementById('script').focus();
                  document.getElementById('script').select();

                  try {
                    document.execCommand('copy');
                  } catch(e) {
                    // ignore
                  }
               }

            //-->
            </script>
         </head>
         <body onload="updateScript()">
            <h1>Authorization procedure</h1>
            <p>In order to get access to the OpenCBDC Test Controller, you will have to generate your own client-side certificate and send the certificate to the administrator of the CDBC Test Controller.</p>
            <p>Fill out the following details, and execute the script at the bottom on your machine to generate a certificate.</p>
            <table>
               <tr><td>Full name:</td><td><input onchange="updateScript()" type="text" id="fullName" placeholder="(ex: John Doe)" /></td></tr>
               <tr><td>E-mail address:</td><td><input onchange="updateScript()" type="text" placeholder="(ex: john.doe@acme.com)" id="email" /></td></tr>
               <tr><td>Organization:</td><td><input onchange="updateScript()" type="text" placeholder="(ex: ACME Software Company)" id="company" /></td></tr>
               <tr><td>Department:</td><td><input onchange="updateScript()" type="text" placeholder="(ex: Development)" id="ou" /></td></tr>
               <tr><td>Locality:</td><td><input onchange="updateScript()" type="text" placeholder="(ex: Boston)" id="locality" /></td></tr>
               <tr><td>State:</td><td><input onchange="updateScript()" type="text" id="state" placeholder="(ex: Massachusetts)"  /></td></tr>
               <tr><td>Country:</td><td><input onchange="updateScript()" type="text" id="country" placeholder="(ex: US)" maxlength="2" /></td></tr>
            </table>
            <p>Execute the following script in a bash shell (<a href="#" onclick="copyScript()">Copy</a>):</p>
            <textarea id="script" style="background-color: #e0e0e0; color: black;" cols=80 rows=12></textarea>
			` + postForm +
		`</body>
      </html>`))
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}

}

func (srv *HttpServer) AuthorizeScriptHandler(
	w http.ResponseWriter,
	req *http.Request,
) {
	w.Header().Add("Content-Type", "text/x-shellscript")
	w.Header().Add("Content-Disposition", "attach; filename=create-cert.sh")
	w.WriteHeader(200)
	_, err := w.Write([]byte(`
      #!/bin/bash
      bold=$(tput bold)
      normal=$(tput sgr0)
      echo -e "Creating a certificate for ${bold}${CERT_CN}${normal}...\n\n"
      echo -e "[req]\nprompt=no\ndefault_bits=4096\nencrypt_key=no\ndefault_md=sha256\ndistinguished_name=req_subj\n[req_subj]\ncommonName=${CERT_CN}\nemailAddress=${CERT_EMAIL}\ncountryName=${CERT_COUNTRY}\nstateOrProvinceName=${CERT_STATE}\nlocalityName=${CERT_LOCALITY}\norganizationName=${CERT_ORG}\norganizationalUnitName=${CERT_OU}\n" > create-cert.conf
      openssl req -x509 -days 365 -newkey rsa:4096 -config $PWD/create-cert.conf -keyout user.key -out user.crt > /dev/null 2>&1
      rm -rf create-cert.conf
      EXPORT_PW=$(openssl rand -hex 16)
      openssl pkcs12 -export -inkey user.key -in user.crt -out user.p12 -password pass:${EXPORT_PW} > /dev/null 2>&1
      rm -rf user.key
      echo -e "\n\n\nYour certificate is ready!\n===\nImport the file ${bold}user.p12${normal} in your browser. Use the password ${bold}${EXPORT_PW}${normal}\n\nFor safety reasons, delete the p12 file once you're done. Provide the ${bold}user.crt${normal} to the CBDC-Test Controller Administrator for authorization. Once the certificate has been authorized for access, you can access the CBDC-Test portal using this client-side certificate.\n\nHave a nice day!.\n\n\n"
   `))
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}

}

func (srv *HttpServer) ReloadCerts() {

	roots := x509.NewCertPool()
	users := []*SystemUser{}
	err := filepath.Walk(
		filepath.Join(common.DataDir(), "certs/users"),
		func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(path, ".crt") {
				caCertPEM, err := ioutil.ReadFile(path)
				if err != nil {
					logging.Warnf(
						"Could not parse certificate %s: %v",
						path,
						err,
					)
					return nil
				}
				ok := roots.AppendCertsFromPEM(caCertPEM)
				if !ok {
					logging.Warnf(
						"Could not parse certificate %s: %v",
						path,
						err,
					)
					return nil
				}

				block, _ := pem.Decode([]byte(caCertPEM))

				if block == nil {
					logging.Warnf(
						"Could not parse certificate %s: %v",
						path,
						err,
					)
					return nil
				}

				cert, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					logging.Warnf(
						"Could not parse certificate %s: %v",
						path,
						err,
					)
					return nil
				}
				usr := srv.UserFromCert(cert)
				usr.certFile = path
				users = append(users, usr)
			}
			return nil
		},
	)
	if err != nil {
		logging.Errorf("Failure loading certs: %v", err)
	}
	srv.roots = roots
	srv.users = users
	srv.certificate, err = tls.LoadX509KeyPair(
		filepath.Join(common.DataDir(), "certs/server.crt"),
		filepath.Join(common.DataDir(), "certs/server.key"),
	)
	if err != nil {
		panic("Failure loading server cert")
	}

}

func (srv *HttpServer) UserFromCert(cert *x509.Certificate) *SystemUser {
	email := ""
	emailASN1 := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}
	if len(cert.EmailAddresses) > 0 {
		email = cert.EmailAddresses[0]
	} else if len(cert.Subject.Names) > 0 {
		for _, n := range cert.Subject.Names {
			if n.Type.Equal(emailASN1) {
				emailVal, ok := n.Value.(string)
				if ok {
					email = emailVal
				}
			}
		}
	}

	org := ""
	if len(cert.Subject.Organization) > 0 {
		org = cert.Subject.Organization[0]
	}

	return &SystemUser{
		CN:           cert.Subject.CommonName,
		Email:        email,
		Organization: org,
		Thumbprint:   fmt.Sprintf("%x", md5.Sum(cert.Raw)),
	}
}

func (srv *HttpServer) GetConfigForClient(
	hi *tls.ClientHelloInfo,
) (*tls.Config, error) {
	return &tls.Config{
		ClientAuth:            tls.RequireAndVerifyClientCert,
		ClientCAs:             srv.roots,
		MinVersion:            tls.VersionTLS12,
		Certificates:          []tls.Certificate{srv.certificate},
		VerifyPeerCertificate: srv.getClientValidator(hi),
	}, nil
}

func (srv *HttpServer) getClientValidator(
	helloInfo *tls.ClientHelloInfo,
) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

		opts := x509.VerifyOptions{
			Roots:         srv.roots,
			CurrentTime:   time.Now(),
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		_, err := verifiedChains[0][0].Verify(opts)
		if err != nil {
			logging.Warnf("Failed certificate verification: %v", err)
		}
		return err
	}
}

func (srv *HttpServer) HealthHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(200)
	_, err := w.Write([]byte(`I'm good, thanks.`))
	if err != nil {
		logging.Errorf("Error writing output: %v", err)
	}
}

func (srv *HttpServer) Run() error {
	go srv.publishToWebsocketsLoop()
	go srv.tokenCleanupLoop()

	r := mux.NewRouter()

	r.HandleFunc("/create-cert.sh", srv.AuthorizeScriptHandler)
	r.HandleFunc("/auth", srv.AuthorizeHandler)
	r.HandleFunc("/health", srv.HealthHandler)
	r.HandleFunc("/", srv.HttpsRedirect)
	r.HandleFunc("/ws/{token}", srv.wsWithTokenHandler)
	r.HandleFunc("/firstTimeAuth", srv.firstTimeAddUserHandler)

	go func() {
		err := http.ListenAndServeTLS(
			fmt.Sprintf(":%d", srv.httpsWithoutClientCertPort),
			filepath.Join(common.DataDir(), "certs/server.crt"),
			filepath.Join(common.DataDir(), "certs/server.key"),
			r,
		)
		if err != nil {
			logging.Errorf("Error on HTTPS server without client cert: %v", err)
		}
	}()
	return srv.srv.ListenAndServeTLS(
		filepath.Join(common.DataDir(), "certs/server.crt"),
		filepath.Join(common.DataDir(), "certs/server.key"),
	)
}

func (srv *HttpServer) GetHttpsEndpoint(
	protocol string,
	port int,
	req *http.Request,
) string {
	target := protocol + "://" + req.Host
	if strings.HasSuffix(
		target,
		fmt.Sprintf(":%d", srv.httpsWithoutClientCertPort),
	) {
		target = target[:strings.LastIndex(target, ":")]
	}
	if strings.HasSuffix(target, fmt.Sprintf(":%d", srv.httpsPort)) {
		target = target[:strings.LastIndex(target, ":")]
	}
	if port != 443 {
		target += fmt.Sprintf(":%d", port)
	}
	return target

}
