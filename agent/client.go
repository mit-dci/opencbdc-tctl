package agent

import (
	"fmt"
	"runtime"

	"github.com/mit-dci/opencbdc-tct/logging"
	"github.com/mit-dci/opencbdc-tct/wire"
)

// RunClient is the main loop for an agent
func (a *Agent) RunClient() {
	// Run the send loop in a separate go routine
	go a.sendLoop()

	// Run the incoming processing of the messages on as much
	// goroutines as we have CPUs
	for i := 0; i < runtime.NumCPU(); i++ {
		go a.processingLoop()
	}

	// Close the processing queue if we exit this method. Most likely
	// reason for exiting is that the connection to the coordinator
	// has been closed. Generally speaking, the scripting that runs the
	// agent binary on an agent machine will do so in a loop, hence exiting
	// this loop will cause a restart of the agent
	defer close(a.processingQueue)
	for {
		// Read a message from the coordinator
		msg, err := a.conn.Recv()
		if err != nil {
			if err.Error() != "EOF" {
				logging.Warnf("Error reading message: %v", err.Error())
			}
			a.conn.Close()
			return
		}
		// Send the message to the processing queue
		a.processingQueue <- msg
	}
}

// processingLoop reads from the processingQueue chan of wire messages and
// processes the message - the result of processing is sent back to the
// coordinator by placing it in the outgoing chan - which is then sent over
// the wire by the sendLoop
func (a *Agent) processingLoop() {
	for msg := range a.processingQueue {
		id := wire.GetMessageHeaderID(msg, "ID")
		returnMsg, err := a.handleMsg(msg)
		if err != nil {
			logging.Errorf("Error handling message %d: %v", id, err)
			returnMsg = &wire.ErrorMsg{Error: err.Error()}
		}
		if returnMsg != nil {
			wire.SetMessageHeaderID(returnMsg, "YourID", id)
			a.outgoing <- returnMsg
		}
	}
}

// sendLoop takes care of reading from the outgoing chan of messages and sending
// them over the wire back to the coordinator
func (a *Agent) sendLoop() {
	for msg := range a.outgoing {
		err := a.conn.Send(msg)
		if err != nil {
			logging.Warnf("Could not send message: %v", err)
			a.conn.Close()
			return
		}
	}
	logging.Warnf("Send loop exited!")
}

// handleMsg is the main entry point for handling messages. Based on the message
// type this method calls underlying handling methods to process the message and
// return its reply to the caller of handleMsg() (processingLoop) which will
// take either the reply message or construct an error message based on the
// returned error and place it in the outgoing queue
func (a *Agent) handleMsg(msg wire.Msg) (wire.Msg, error) {
	var err error
	var reply wire.Msg

	// Based on the message type, call the appropriate handler
	switch t := msg.(type) {
	case *wire.PrepareEnvironmentRequestMsg:
		reply, err = a.handlePrepareEnvironment(t)
	case *wire.DestroyEnvironmentMsg:
		reply, err = a.handleDestroyEnvironment(t)
	case *wire.DeployFileRequestMsg:
		reply, err = a.handleDeployFile(t)
	case *wire.DeployFileFromS3RequestMsg:
		reply, err = a.handleDeployFileFromS3(t)
	case *wire.RenameFileRequestMsg:
		reply, err = a.handleRenameFile(t)
	case *wire.ExecuteCommandRequestMsg:
		reply, err = a.handleExecuteCommand(t)
	case *wire.UploadFileToS3RequestMsg:
		reply, err = a.handleUploadFileToS3(t)
	case *wire.BreakCommandRequestMsg:
		reply, err = a.handleBreakCommand(t)
	case *wire.TerminateCommandRequestMsg:
		reply, err = a.handleTerminateCommand(t)
	case *wire.PingMsg:
		reply, err = &wire.AckMsg{}, nil
	case *wire.AckMsg:
		reply, err = nil, nil
	case *wire.ErrorMsg:
		logging.Errorf("Received error from controller: %v", t.Error)
		reply, err = nil, nil
	default:
		reply, err = nil, fmt.Errorf("unknown message type %T", t)
	}

	// If there was an error, send that
	if err != nil {
		return nil, err
	}

	// If there was a reply, send that
	if reply != nil {
		return reply, nil
	}

	// No reply and no error - there will be no response to this message
	return nil, nil
}
