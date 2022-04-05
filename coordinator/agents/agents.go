package agents

import (
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tctl/coordinator"
	"github.com/mit-dci/opencbdc-tctl/coordinator/sources"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// AgentsManager contains easy functions to interact with a connected agent
type AgentsManager struct {
	coord          *coordinator.Coordinator
	src            *sources.SourcesManager
	ev             chan coordinator.Event
	commandDetails sync.Map
}

// NewAgentsManager creates a new AgentsManager
func NewAgentsManager(
	c *coordinator.Coordinator,
	src *sources.SourcesManager,
	ev chan coordinator.Event,
) (*AgentsManager, error) {
	return &AgentsManager{
		coord:          c,
		src:            src,
		commandDetails: sync.Map{},
		ev:             ev,
	}, nil
}

// QueryAgentWithTimeout is a utility function to do a single request-response
// interaction with the given agent. It will wait for a response for up to
// the specified timeout, and return an error if the response was not received
// within that timeout.
func (am *AgentsManager) QueryAgentWithTimeout(
	agentID int32,
	msg wire.Msg,
	timeout time.Duration,
) (wire.Msg, error) {
	rc := make(chan wire.Msg, 1)
	err := am.coord.SendToAgent(agentID, msg, rc)
	if err != nil {
		return nil, err
	}
	return wire.ReceiveWithTimeout(rc, timeout)
}

// QueryAgent is a utility function to do a single request-response interaction
// with the given agent. It will wait for a response for up to 15 seconds, and
// return an error if the response was not received within that timeout.
func (am *AgentsManager) QueryAgent(
	agentID int32,
	msg wire.Msg,
) (wire.Msg, error) {
	return am.QueryAgentWithTimeout(agentID, msg, time.Second*15)
}
