package coordinator

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

var ErrAgentNotFound = errors.New("agent not found")

// Coordinator is the main type that manages the connections
// to the test agents
type Coordinator struct {
	server      *wire.Listener    // The main listener
	nextAgentID int32             // The indexer for agents
	agents      []*ConnectedAgent // The array of connected agents
	agentsLock  sync.Mutex        // Lock guarding the agents array
	events      chan Event        // The channel for real-time (websocket)info
	maintenance bool              // The current state of the maintenance mode
}

// ConnectedAgent holds the information for a currently connected test agent
type ConnectedAgent struct {
	// The ID for the connected agent
	ID int32 `json:"id"`
	// The underlying wire connection to this agent
	conn *wire.Conn
	// The channel to send messages to the agent
	outgoing chan wire.Msg
	// Indicates if the agent has completed its handshake with the coordinator
	handshakeComplete bool
	// Most recently received system information for this agent
	SystemInfo common.AgentSystemInfo `json:"systemInfo"`
	// The binary version of the agent binary that connected
	AgentVersion string `json:"agentVersion"`
	// The current ping roundtrip time as measured from the coordinator
	PingRTT float64 `json:"pingRTT"`
	// The array of registered listeners that are expecting reply or update
	// messages
	listeners []*agentReplyListener
	// The lock guarding listeners
	listenersLock sync.Mutex
	// Indicates if this connection is (being) closed
	closed bool
	// The lock guarding closed
	closeLock sync.Mutex
}

// This type describes a listener for messages from the agent. Various parts
// of the coordinator logic can require listening to updates from the agent,
// which can either be reply messages to a specific message sent by us (matched
// on ID) or it can be streaming updates for a particular running command
// (matched on commandID)
type agentReplyListener struct {
	// The channel to which the subscriber is listening for update(s)
	replyChan chan wire.Msg
	// The ID of the message we sent that we're awaiting a reply for
	// (or -1 if it is a command listener)
	ourID int
	// The command ID we are awaiting updates for (or nil if it is a
	// listener for a reply message)
	commandID []byte
}

// NewCoordinator creates a new instance of the Coordinator type, listening on
// the port identified by the `port` argument, and using the `ev` argument to
// send real-time updates to. Returns an error if unable to listen for new
// connections
func NewCoordinator(ev chan Event, port int) (*Coordinator, error) {
	srv, err := wire.NewServer(port)
	if err != nil {
		return nil, err
	}
	return &Coordinator{
		server:     srv,
		agents:     []*ConnectedAgent{},
		agentsLock: sync.Mutex{},
		events:     ev,
	}, nil
}

// RunServer is the main loop for the endpoint that agents connect to - it will
// wait for new connections and then handle those in a separate goroutine.
func (c *Coordinator) RunServer() error {
	for {
		// Accept a new connection from an agent
		clt, err := c.server.Accept()
		if err != nil {
			return err
		}

		// Create a new ConnectedAgent struct and append it to the array
		// of active connected agents
		connectedAgent := ConnectedAgent{
			conn:          clt,
			ID:            atomic.AddInt32(&c.nextAgentID, 1),
			outgoing:      make(chan wire.Msg, 100),
			listenersLock: sync.Mutex{},
			listeners:     []*agentReplyListener{},
		}
		clt.Tag = fmt.Sprintf("Agent %d", connectedAgent.ID)
		c.addAgent(&connectedAgent)

		// Process the incoming messages (from the agent) in a new goroutine
		go c.handleConn(&connectedAgent)
		// Process sending outgoing messages (to the agent) in a new goroutine
		go connectedAgent.sendLoop()
		// Start a separate loop that pings the agent to see if the connection
		// remains alive, and what the roundtrip time on the TCP connection is
		go c.pingLoop(&connectedAgent)
	}
}

// addAgent appends a new ConnectedAgent to the array of connected agents, using
// the agentsLock as guard. It will also send an update to the real-time channel
// with the new count of connected agents.
func (c *Coordinator) addAgent(agent *ConnectedAgent) {
	newLen := 0

	// Acquire lock and append the agent to the array
	c.agentsLock.Lock()
	c.agents = append(c.agents, agent)
	newLen = len(c.agents)
	c.agentsLock.Unlock()

	// Send the updated connected agent count to the
	// real time event channel
	c.events <- Event{
		Type: EventTypeConnectedAgentCountChanged,
		Payload: ConnectedAgentCountChangedPayload{
			Count: newLen,
		},
	}
}

// removeAgent removes the ConnectedAgent from the array of connected agents,
// using the agentsLock as guard. It will also send an update to the real-time
// channel with the new count of connected agents.
func (c *Coordinator) removeAgent(agent *ConnectedAgent) {
	newLen := 0

	// Acquire lock and remove the agent from the array
	c.agentsLock.Lock()
	newAgents := make([]*ConnectedAgent, 0)
	for _, a := range c.agents {
		if a.ID != agent.ID {
			newAgents = append(newAgents, a)
		}
	}
	c.agents = newAgents
	newLen = len(c.agents)
	c.agentsLock.Unlock()

	// Send the updated connected agent count to the
	// real time event channel
	c.events <- Event{
		Type: EventTypeConnectedAgentCountChanged,
		Payload: ConnectedAgentCountChangedPayload{
			Count: newLen,
		},
	}
}

// GetAgent returns the ConnectedAgent instance referenced by the passed agentID
// Returns ErrAgentNotFound if there is no agent with the given agentID
func (c *Coordinator) GetAgent(agentID int32) (*ConnectedAgent, error) {
	for _, a := range c.agents {
		if a.ID == agentID {
			return a, nil
		}
	}
	return nil, ErrAgentNotFound
}

// GetAgents returns a copy of the slice of all agents that are currently
// connected
func (c *Coordinator) GetAgents() []*ConnectedAgent {
	return c.agents[:]
}

// GetAgentCount returns the currently connected number of agents
func (c *Coordinator) GetAgentCount() int {
	return len(c.agents)
}

// SendToAgent send a wire message to an agent, and registers a listener for the
// reply to that message's ID that will deliver the response to replyChan
func (c *Coordinator) SendToAgent(
	agentID int32,
	msg wire.Msg,
	replyChan chan wire.Msg,
) error {
	a, err := c.GetAgent(agentID)
	if err != nil {
		return err
	}
	if replyChan != nil {
		a.conn.SetMessageID(msg)
		l := agentReplyListener{
			ourID:     wire.GetMessageHeaderID(msg, "ID"),
			replyChan: replyChan,
		}
		logging.Debugf(
			"Registered message callback for agent %d, message %d with channel %v",
			agentID,
			l.ourID,
			replyChan,
		)
		a.listenersLock.Lock()
		a.listeners = append(a.listeners, &l)
		a.listenersLock.Unlock()
	}
	return a.sendMsg(msg)
}

// RegisterCommandStatusCallback will register a listener for all update
// messages
// that pertain to the command specified by commandID, running on the agent
// specified by agentID and will deliver those updates to replyChan
func (c *Coordinator) RegisterCommandStatusCallback(
	agentID int32,
	commandID []byte,
	replyChan chan wire.Msg,
) error {
	a, err := c.GetAgent(agentID)
	if err != nil {
		return err
	}
	logging.Debugf(
		"Registered command callback for agent %d, command %x with channel %v",
		agentID,
		commandID,
		replyChan,
	)
	l := agentReplyListener{
		ourID:     -1, /* will never match */
		commandID: commandID,
		replyChan: replyChan,
	}
	a.listenersLock.Lock()
	a.listeners = append(a.listeners, &l)
	a.listenersLock.Unlock()
	return nil
}

// handleConn is responsible for handling a single connected agent's incoming
// messages, calling handleMsg() on them and send the result of handling the
// message back to the agent using sendMsg()
func (c *Coordinator) handleConn(agent *ConnectedAgent) {
	for {
		// Read the next message from the connection
		msg, err := agent.conn.Recv()
		if err != nil {
			// If something goes wrong, close the connection
			// and remove the agent. If the error is not EOF
			// (which indicates the remote site terminated the
			// connection), log whatever went wrong as well
			if err.Error() != "EOF" {
				logging.Warnf("Error reading message: %v", err.Error())
			}
			agent.conn.Close()
			c.removeAgent(agent)
			return
		}
		// Read the ID from the incoming message
		id := wire.GetMessageHeaderID(msg, "ID")
		// Handle the message
		returnMsg, err := c.handleMsg(agent, msg)
		// If an error occurred handling the message, return an
		// ErrorMsg in stead
		if err != nil {
			returnMsg = &wire.ErrorMsg{Error: err.Error()}
			logging.Warnf(
				"Error handling message [%T] from agent %d: %v",
				msg,
				agent.ID,
				err.Error(),
			)
		}
		if returnMsg != nil {
			// Set the YourID on the message header to the ID of the incoming
			// message to indicate that we are responding to that
			wire.SetMessageHeaderID(returnMsg, "YourID", int(id))
			// Send our reply or error back to the agent
			err = agent.sendMsg(returnMsg)
			if err != nil {
				// if we were unable to send, remove the agent. The only
				// error that can occur in sendMsg is that the agent was
				// closed already - in which case we shouldn't call close()
				// again
				c.removeAgent(agent)
				return
			}
		}
	}
}

// handleMsg is the main entry point for handling a message. It will dispatch
// the message to the correct handling function based on its type
func (c *Coordinator) handleMsg(
	agent *ConnectedAgent,
	msg wire.Msg,
) (wire.Msg, error) {
	var err error
	var reply wire.Msg

	switch t := msg.(type) {
	case *wire.HelloMsg:
		reply, err = c.handleHello(agent, t)
	case *wire.UpdateSystemInfoMsg:
		reply, err = c.handleUpdateSystemInfo(agent, t)
	default:
		// Check if someone's waiting for the reply
		repliedToID := wire.GetMessageHeaderID(t, "YourID")
		newListeners := make([]*agentReplyListener, 0)
		sentReply := false
		cmdStatus, isCmdStatus := msg.(*wire.ExecuteCommandStatusMsg)
		agent.listenersLock.Lock()
		defer agent.listenersLock.Unlock()
		for _, rl := range agent.listeners {
			if rl.ourID == repliedToID {
				// This message is a reply to one we sent before and there is
				// a registered listener for this reply. Send it to the channel
				// corresponding to the listener (non-blocking, time out after
				// a second). This timeout is to prevent a full channel from
				// causing this logic to hang, which blocks the connection to
				// this agent entirely
				select {
				case rl.replyChan <- msg:
					break
				case <-time.After(time.Second * 1):
					logging.Warnf("Timeout delivering message to channel %v for reply on %d", rl.replyChan, rl.ourID)
				}
				sentReply = true
				// We're not adding this listener back to the newListeners array
				// because we only want a single reply to a command
			} else if isCmdStatus && bytes.Equal(rl.commandID, cmdStatus.CommandID) {
				// This message is a command status update for a command that
				// this listener is registered to listen to. Send it to the
				// channel corresponding to the listener (non-blocking, time out
				// after a second). This timeout is to prevent a full channel
				// from causing this logic to hang, which blocks the connection
				// to this agent entirely
				select {
				case rl.replyChan <- msg:
					break
				case <-time.After(time.Second * 1):
					logging.Warnf("Timeout delivering message to channel %v for command %x", rl.replyChan, rl.commandID)
				}
				sentReply = true
				if cmdStatus.Status != wire.CommandStatusFinished {
					// If the command's status is not finished, we expect
					// further updates and thus re-register the listener for
					// the next update message
					newListeners = append(newListeners, rl)
				}
			} else {
				// This listener was irrelevant for this message, so we will
				// need to continue using it for matching future messages
				newListeners = append(newListeners, rl)
			}
		}

		if sentReply {
			// If we sent this update to one of our listeners, we have to apply
			// the new listener array from which either command-response
			// listeners
			// that got a reply were removed, or command-status listeners for
			// commands that finished are removed
			agent.listeners = newListeners

			return nil, nil
		}

		// Ignore unneeded acks - these are Acks that we got from the agent
		// that we did not listen for in a reply listener. We can just ignore
		// these.
		_, isAck := msg.(*wire.AckMsg)
		if isAck {
			return nil, nil
		}

		// If we received a message that is no reply, no hello or systemInfo
		// update, no command status update and no ack, then we don't know how
		// to process it - but it cannot just be ignored. We should return an
		// error so that handleConn will return an ErrorMsg over the wire
		err = fmt.Errorf("unable to process message of type %T", t)
	}

	if err != nil {
		return nil, err
	}
	if reply != nil {
		return reply, nil
	}
	return &wire.AckMsg{}, nil
}

// handleHello handles the initial handshake from the agent and sets some
// additional metadata about the agent (system info and agent version)
func (c *Coordinator) handleHello(
	agent *ConnectedAgent,
	msg *wire.HelloMsg,
) (wire.Msg, error) {
	agent.SystemInfo = msg.SystemInfo
	agent.AgentVersion = msg.AgentVersion
	agent.handshakeComplete = true
	return &wire.HelloResponseMsg{YourAgentID: agent.ID}, nil
}

// handleUpdateSystemInfo will update the known system information for a
// connected agent, when the agent sends an UpdateSystemInfoMsg
func (c *Coordinator) handleUpdateSystemInfo(
	agent *ConnectedAgent,
	msg *wire.UpdateSystemInfoMsg,
) (wire.Msg, error) {
	agent.SystemInfo = msg.SystemInfo
	return nil, nil
}

// pingLoop send a PingMsg to the connected agent every 30 seconds and records
// the time needed to get the Ack message back. If there is no reply for five
// seconds, we record a no-reply. If this happens three times in a row, we
// consider the agent dead and disconnect it.
func (c *Coordinator) pingLoop(a *ConnectedAgent) {
	noReplyCount := 0
	for {
		time.Sleep(time.Second * 30)
		rc := make(chan wire.Msg, 1)
		start := time.Now()
		err := c.SendToAgent(a.ID, &wire.PingMsg{}, rc)
		if err != nil {
			if err == ErrAgentNotFound {
				// Already gone
				return
			}
			logging.Errorf("Error sending ping to agent: %v", err)
			a.close()
			c.removeAgent(a)
			return
		}

		select {
		case <-rc:
			a.PingRTT = float64(
				time.Since(start).Nanoseconds(),
			) / float64(
				1000000,
			)
			noReplyCount = 0
			break
		case <-time.After(5 * time.Second):
			noReplyCount++
			if noReplyCount > 3 { // Disconnect agent - no reply for 3 consecutive pings
				a.close()
				c.removeAgent(a)
				return
			}
			a.PingRTT = -1
		}
	}
}

// sendMsg places the msg argument in the queue for sending to the agent. First
// the sendMsg method checks if the connection is not yet (being) closed and
// returns an error if that is the case.
func (a *ConnectedAgent) sendMsg(msg wire.Msg) (err error) {
	a.closeLock.Lock()
	defer a.closeLock.Unlock()
	if a.closed {
		return errors.New("chan is closed")
	}
	a.outgoing <- msg
	return nil
}

// close closes the agent after acquiring a lock on the closeLock - it will also
// close the outgoing channel to end the sendLoop
func (a *ConnectedAgent) close() {
	a.closeLock.Lock()
	a.conn.Close()
	if !a.closed {
		close(a.outgoing)
		a.closed = true
	}
	a.closeLock.Unlock()
}

// sendLoop is responsible for sending the messages in the outgoing channel to
// the wire. It will exit when there have been no messages for 20 minutes.
func (a *ConnectedAgent) sendLoop() {
	for {
		select {
		case msg := <-a.outgoing:
			err := a.conn.Send(msg)
			if err != nil {
				logging.Infof(
					"Could not send message to agent %d: %v",
					a.ID,
					err,
				)
				a.close()
				return
			}
		case <-time.After(20 * time.Minute):
			logging.Infof(
				"Did not receive message from agent %d for 20 minutes, closing",
				a.ID,
			)
			a.close()
			return
		}
	}
}

// SetMaintenance sets the current maintenance mode state of the system. If the
// system is in maintenance mode, test runs will not be executed. You can queue
// new tests, but they will not be run until the maintenance mode is turned off.
// Maintenance mode can be handy when deploying new versions of the test
// controller through the build pipeline, as (re)deploying the controller in
// the middle of a test run will terminate and invalidate that testrun.
func (c *Coordinator) SetMaintenance(maint bool) {
	c.maintenance = maint
	c.events <- Event{
		Type: EventTypeMaintenanceModeChanged,
		Payload: MaintenanceModeChangedPayload{
			MaintenanceMode: c.maintenance,
		},
	}
}

// GetMaintenance returns the current maintenance mode status of the system
func (c *Coordinator) GetMaintenance() bool {
	return c.maintenance
}
