package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/mit-dci/opencbdc-tct/coordinator"
	"github.com/mit-dci/opencbdc-tct/coordinator/agents"
	"github.com/mit-dci/opencbdc-tct/coordinator/awsmgr"
	"github.com/mit-dci/opencbdc-tct/coordinator/http"
	"github.com/mit-dci/opencbdc-tct/coordinator/scripts"
	"github.com/mit-dci/opencbdc-tct/coordinator/sources"
	"github.com/mit-dci/opencbdc-tct/coordinator/testruns"
	"github.com/mit-dci/opencbdc-tct/logging"
)

var GitCommit string
var BuildDate string

func main() {
	logging.SetLogLevel(int(logging.LogLevelInfo))
	coordinatorPort, httpPort := getPorts()

	ev := make(chan coordinator.Event, 10000)

	if os.Getenv("S3_INTERFACE_ENDPOINT") == "" {
		logging.Infof(
			"S3_INTERFACE_ENDPOINT not set, S3 will default to public endpoints",
		)
	}
	if os.Getenv("S3_INTERFACE_REGION") == "" {
		logging.Infof(
			"S3_INTERFACE_REGION not set, S3 will default to public endpoints",
		)
	}

	logging.Infof("Creating coordinator")

	c, err := coordinator.NewCoordinator(ev, coordinatorPort)
	if err != nil {
		panic(err)
	}

	logging.Infof("Extracting scripts")
	err = scripts.WriteScripts()
	if err != nil {
		panic(err)
	}

	logging.Infof("Creating sources manager")

	s := sources.NewSourcesManager()

	logging.Infof("Creating agents manager")

	am, err := agents.NewAgentsManager(c, s, ev)
	if err != nil {
		panic(err)
	}

	logging.Infof("Creating AWS manager")

	awsm := awsmgr.NewAwsManager()

	logging.Infof("Creating TestRun manager")
	tr, err := testruns.NewTestRunManager(c, am, s, ev, awsm, GitCommit)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			err := s.EnsureSourcesUpdated()
			if err != nil {
				logging.Errorf("Sources could not be updated: %v", err)
			}
			time.Sleep(time.Minute * 5)
		}
	}()

	go tr.LoadAllTestRuns()

	logging.Infof("Creating HTTP Server")

	chttp, err := http.NewHttpServer(
		c,
		s,
		am,
		tr,
		ev,
		awsm,
		httpPort,
		fmt.Sprintf("%s-%s", BuildDate, GitCommit[:7]),
	)
	if err != nil {
		panic(err)
	}

	go func() {
		logging.Infof("Starting HTTP server")
		err := chttp.Run()
		if err != nil {
			panic(err)
		}
	}()

	logging.Infof("Starting Coordinator")

	err = c.RunServer()
	if err != nil {
		panic(err)
	}
}

func getPorts() (int, int) {
	coordinatorPort, _ := strconv.Atoi(os.Getenv("PORT"))
	if coordinatorPort == 0 {
		coordinatorPort = 8000
	}

	httpPort, _ := strconv.Atoi(os.Getenv("PORT"))
	if httpPort == 0 {
		httpPort = 8080
	}

	return coordinatorPort, httpPort
}
