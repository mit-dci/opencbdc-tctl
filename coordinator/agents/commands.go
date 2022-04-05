package agents

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/coordinator"
	"github.com/mit-dci/opencbdc-tctl/logging"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// RunningCommandsForAgent returns a list of commands that are currently running
// on the agent specified by agentID
func (am *AgentsManager) RunningCommandsForAgent(
	agentID int32,
) []coordinator.AgentCommandRunningPayload {
	retVal := []coordinator.AgentCommandRunningPayload{}
	am.commandDetails.Range(func(k, v interface{}) bool {
		det, ok := v.(coordinator.AgentCommandRunningPayload)
		if ok && det.AgentID == agentID {
			retVal = append(retVal, det)
		}
		return true
	})
	return retVal
}

// ExecuteCommand will execute a command on the agent specified by the agentID.
// It will run the command specified in `command`, with the parameters and
// environment variables specified by the `params` and `env` arguments. It will
// execute the command in the environment specified by `environmentID`.
// The working directory will be set to the `dir` parameter, which is relative
// to the root of the enviroment's working directory. `timeout` indicates how
// long the command should be allowed to run (in seconds). `wait` indicates if
// the function call should block until the command is completed. `profile`
// determines if we gather system performance metrics while running the command.
// `perfProfile` determines if we run `perf` performance profiling on the
// process and `perfSampleRate` the samples per second that we have `perf`
// gather. `debug` determines if we run the command in gdb for debugging.
// `commandResults` is a channel where we are supposed to report the command's
// results once the agent has completed it.
func (am *AgentsManager) ExecuteCommand(
	agentID int32,
	command string,
	params, env []string,
	environmentID []byte,
	dir string,
	timeout int,
	commandResults chan *common.ExecutedCommand,
	wait bool,
	profile bool,
	perf bool,
	perfSampleRate int,
	debug bool,
	recordNetwork bool,
) ([]byte, error) {

	// Send the ExecuteCommandRequestMsg to the agent and get its
	// reply
	msg, err := am.QueryAgent(agentID, &wire.ExecuteCommandRequestMsg{
		EnvironmentID:        environmentID,
		Dir:                  dir,
		Env:                  env,
		Command:              command,
		Parameters:           params,
		Profile:              profile,
		PerfProfile:          perf,
		PerfSampleRate:       perfSampleRate,
		Debug:                debug,
		S3OutputRegion:       os.Getenv("AWS_REGION"),
		S3OutputBucket:       os.Getenv("OUTPUTS_S3_BUCKET"),
		RecordNetworkTraffic: recordNetwork,
	})
	if err != nil {
		return nil, err
	}

	// Check if the reply is of the valid type and that it was successful
	rep, ok := msg.(*wire.ExecuteCommandResponseMsg)
	if !ok {
		return nil, fmt.Errorf(
			"expected ExecuteCommandResponseMsg, got %T",
			rep,
		)
	}
	if !rep.Success {
		return nil, fmt.Errorf(
			"error starting command %s script: %s",
			command,
			rep.Error,
		)
	}

	// Convert the commandID to a string
	cmdIDStr := fmt.Sprintf("%x", rep.CommandID)

	// Store the running command in our cache of running commands
	details := coordinator.AgentCommandRunningPayload{
		AgentID:     agentID,
		CommandID:   cmdIDStr,
		Command:     command,
		Params:      params,
		Environment: env,
		Started:     time.Now(),
	}
	am.commandDetails.Store(cmdIDStr, details)

	if wait { // Wait inline for the command to complete
		return rep.CommandID, am.waitForCommandFinish(
			agentID,
			rep.CommandID,
			timeout,
			commandResults,
		)
	}

	// IF not waiting inline, still run waitForCommandFinish in a goroutine to
	// follow the progress
	// through the commandResults chan and fire events
	go func() {
		err := am.waitForCommandFinish(
			agentID,
			rep.CommandID,
			timeout,
			commandResults,
		)
		if err != nil {
			logging.Warnf(
				"waitForCommandFinish failed for command %x on agent %d: %v",
				rep.CommandID,
				agentID,
				err,
			)
		}
	}()
	return rep.CommandID, nil
}

// waitForCommandFinish will register a callback with the coordinator to receive
// update messages sent by the agent about the progress of the command. It will
// then listen for updates on that channel and error out if the command takes
// too long, or no updates have been received for more than 30 seconds. Once
// the agent reports the command finished, we will inform the ``
func (am *AgentsManager) waitForCommandFinish(
	agentID int32,
	commandID []byte,
	timeout int,
	commandResults chan *common.ExecutedCommand,
) error {
	// Make a channel for updates on the command
	rc := make(chan wire.Msg, 100)
	// Register the listener in the coordinator to send updates
	// from the agent for this command to the given channel
	err := am.coord.RegisterCommandStatusCallback(agentID, commandID, rc)
	if err != nil {
		return err
	}
	cmdIDStr := fmt.Sprintf("%x", commandID)
	start := time.Now()
	for {
		// Check for the command timeout
		if time.Since(start).Seconds() > float64(timeout) {
			return fmt.Errorf("command timed out")
		}

		// Fetch the next command status update. The agent sends a status
		// update for running commands every 20 seconds, so it should never
		// take more than 30 seconds to receive, unless something's wrong.
		msg, err := wire.ReceiveWithTimeout(rc, time.Second*120)
		if err != nil {
			return fmt.Errorf("did not receive status update: %s", err.Error())
		}
		rep, ok := msg.(*wire.ExecuteCommandStatusMsg)
		if !ok {
			return fmt.Errorf(
				"did not receive status update while waiting for - type wrong",
			)
		}

		// When the command is finished, report its details and return code
		// to the commandResults channel (if it's nil) and delete the command
		// from the commandDetails map
		if rep.Status == wire.CommandStatusFinished {
			if commandResults != nil {
				detailsRaw, ok := am.commandDetails.Load(cmdIDStr)
				if ok {
					details, ok := detailsRaw.(coordinator.AgentCommandRunningPayload)
					if ok {
						commandResults <- &common.ExecutedCommand{
							Description: details.Command,
							Params:      details.Params,
							Environment: details.Environment,
							ExitCode:    rep.ExitCode,
							AgentID:     agentID,
							CommandID:   cmdIDStr,
						}
					}
				}
			}
			am.commandDetails.Delete(cmdIDStr)
			return nil
		}

	}
}

// BreakCommand will instruct the agent to break the given command by sending
// it a os.Interrupt signal
func (am *AgentsManager) BreakCommand(agentID int32, commandID []byte) error {
	msg, err := am.QueryAgentWithTimeout(agentID, &wire.BreakCommandRequestMsg{
		CommandID: commandID,
	}, time.Minute)
	if err != nil {
		return err
	}
	_, ok := msg.(*wire.AckMsg)
	if !ok {
		errMsg, ok := msg.(*wire.ErrorMsg)
		if ok {
			return errors.New(errMsg.Error)
		}
		return common.ErrWrongMessageType
	}
	return nil
}

// TerminateCommand will instruct the agent to terminate the given command by
// sending it a os.Kill signal
func (am *AgentsManager) TerminateCommand(
	agentID int32,
	commandID []byte,
) error {
	msg, err := am.QueryAgentWithTimeout(
		agentID,
		&wire.TerminateCommandRequestMsg{
			CommandID: commandID,
		},
		time.Minute,
	)
	if err != nil {
		return err
	}
	_, ok := msg.(*wire.AckMsg)
	if !ok {
		errMsg, ok := msg.(*wire.ErrorMsg)
		if ok {
			return errors.New(errMsg.Error)
		}
		return common.ErrWrongMessageType
	}
	return nil
}
