package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mit-dci/cbdc-test-controller/agent"
	"github.com/mit-dci/cbdc-test-controller/agent/scripts"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

// These two variables are set while building, see the Dockerfile.agent
// If not specified, they default to today and "dev"
var GitCommit string
var BuildDate string

func main() {
	logging.SetLogLevel(int(logging.LogLevelInfo))
	version := fmt.Sprintf("%s-%s", time.Now().Format("20060102"), "dev")
	if GitCommit != "" && BuildDate != "" {
		version = fmt.Sprintf("%s-%s", BuildDate, GitCommit[:7])
	}
	logging.Infof("Agent v%s booting...", version)

	iface, err := agent.GetNetworkInterfaceName()
	if err != nil {
		logging.Errorf("Cannot determine primary network interface: %v", err)
		os.Exit(127)
	}
	logging.Infof("My primary network interface is %s", iface)

	err = scripts.WriteScripts()
	if err != nil {
		logging.Errorf(
			"Failed to extract scripts: [%s], exiting...\n",
			err.Error(),
		)
		os.Exit(128)
	}

	// Parse the coordinator endpoint from either the command line flags
	// or the environment
	host := ""
	port := 0
	flag.StringVar(&host, "host", "", "Coordinator host to connect to")
	flag.IntVar(&port, "port", 0, "Coordinator port to connect to")
	flag.Parse()
	if host == "" {
		host = os.Getenv("COORDINATOR_HOST")
		port, _ = strconv.Atoi(os.Getenv("COORDINATOR_PORT"))
	}
	if port == 0 {
		port = 8000
	}
	if os.Getenv("S3_INTERFACE_ENDPOINT") == "" {
		logging.Infof("S3_INTERFACE_ENDPOINT not set, S3 will default to public endpoints")
	}
	if os.Getenv("S3_INTERFACE_REGION") == "" {
		logging.Infof("S3_INTERFACE_REGION not set, S3 will default to public endpoints")
	}

	// Connect the agent to the coordinator
	logging.Infof("Connecting to server %s on port %d...\n", host, port)
	a, err := agent.NewAgent(version, host, port)
	if err != nil {
		logging.Errorf("Failed to connect: [%s], exiting...\n", err.Error())
		os.Exit(129)
	}

	// Synchronize the time with an NTP server such that the agents are in sync
	// as much as possible
	a.SyncTime()

	// Create a .debug directory in the user's home folder to store the output
	// of running perf
	err = os.MkdirAll(filepath.Join(os.Getenv("HOME"), ".debug"), 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		logging.Errorf(
			"Failed to create .debug directory: [%s], exiting...\n",
			err.Error(),
		)
		os.Exit(130)
	}

	// Execute the main agent loop
	a.RunClient()

	logging.Infof("Agent loop complete, shutting down")
}
