package agent

import (
	"fmt"
	"os/exec"
	"sync"

	"github.com/mit-dci/opencbdc-tctl/logging"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// Agent is the main class for the agent binary
type Agent struct {
	// The connection to the coordinator
	conn *wire.Conn
	// The queue for incoming messages to process
	processingQueue chan wire.Msg
	// The queue for outgoing messages to send
	outgoing chan wire.Msg
	// The agent's version - injected by the main binary in cmd/agent/main.go
	version string
	// The list of commands running on the agent
	pendingCommands []*pendingCommand
	// The lock for pendingCommands
	pendingCommandsLock sync.Mutex
}

// pendingCommand describes a command that is currently being executed
type pendingCommand struct {
	// Randomly generated ID for each command
	id []byte
	// The underlying process that's being executed
	cmd *exec.Cmd
}

// NewAgent creates a new instance of the Agent class. Requires injection of the
// version number from the main binary, as well as the coordinator's host and
// port to connect to
func NewAgent(
	version string,
	coordinatorHost string,
	coordinatorPort int,
) (*Agent, error) {
	// Create new wire client to connect to the coordinator
	clt, err := wire.NewClient(coordinatorHost, coordinatorPort)
	if err != nil {
		return nil, err
	}

	// Create a new instance of the Agent
	a := &Agent{
		version:             version,
		conn:                clt,
		processingQueue:     make(chan wire.Msg, 100),
		outgoing:            make(chan wire.Msg, 100),
		pendingCommands:     []*pendingCommand{},
		pendingCommandsLock: sync.Mutex{},
	}

	// Send a Hello message to the coordinator to initiate
	// handshake
	msg := a.composeHello()
	err = clt.Send(msg)
	if err != nil {
		return nil, err
	}
	sentID := wire.GetMessageHeaderID(msg, "ID")

	// Await a message reply to our handshake
	reply, err := clt.Recv()
	if err != nil {
		return nil, err
	}

	// Check if the reply is a valid response to our handshake
	ack := false
	switch t := reply.(type) {
	case *wire.HelloResponseMsg:
		ack = (t.Header.YourID == sentID)
		clt.Tag = fmt.Sprintf("Agent %d", t.YourAgentID)
		logging.Debugf("We are agent ID %d on the coordinator", t.YourAgentID)
	case *wire.ErrorMsg:
		return nil, fmt.Errorf("Handshake failed: %v", t.Error)
	}

	if !ack {
		return nil, fmt.Errorf("Handshake failed: no ack")
	}

	// Start the loop that will periodically send the system info
	// to the coordinator
	go a.updateSystemInfoLoop()

	return a, nil
}

// composeHello creates a new wire.HelloMsg with the current system information
// and agent version
func (a *Agent) composeHello() *wire.HelloMsg {
	return &wire.HelloMsg{
		SystemInfo:   GetSystemInfo(),
		AgentVersion: a.version,
	}
}
