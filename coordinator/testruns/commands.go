package testruns

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/coordinator"
)

// runningCommand is used to store a reference to all active commands' IDs and
// the agent they're running on
type runningCommand struct {
	agentID   int32
	commandID []byte
}

// roleBinaries is a map from the system role to the location of the executable
// to run in the binaries archive
var roleBinaries = map[common.SystemRole]string{
	common.SystemRoleArchiver:              "sources/build/src/uhs/atomizer/archiver/archiverd",
	common.SystemRoleRaftAtomizer:          "sources/build/src/uhs/atomizer/atomizer/atomizer-raftd",
	common.SystemRoleSentinel:              "sources/build/src/uhs/atomizer/sentinel/sentineld",
	common.SystemRoleShard:                 "sources/build/src/uhs/atomizer/shard/shardd",
	common.SystemRoleCoordinator:           "sources/build/src/uhs/twophase/coordinator/coordinatord",
	common.SystemRoleWatchtower:            "sources/build/src/uhs/atomizer/watchtower/watchtowerd",
	common.SystemRoleShardTwoPhase:         "sources/build/src/uhs/twophase/locking_shard/locking-shardd",
	common.SystemRoleSentinelTwoPhase:      "sources/build/src/uhs/twophase/sentinel_2pc/sentineld-2pc",
	common.SystemRoleAtomizerCliWatchtower: "sources/build/tools/bench/atomizer-cli-watchtower",
	common.SystemRoleTwoPhaseGen:           "sources/build/tools/bench/twophase-gen",
	common.SystemRoleAgent:                 "sources/build/src/parsec/agent/agentd",
	common.SystemRoleRuntimeLockingShard:   "sources/build/src/parsec/runtime_locking_shard/runtime_locking_shardd",
	common.SystemRoleTicketMachine:         "sources/build/src/parsec/ticket_machine/ticket_machined",
	common.SystemRolePhaseTwoGen:           "sources/build/tools/bench/parsec/evm/evm_bench",
}

// roleParameters is a map from the system role to the parameters we have to
// pass to the binary when running it. We use placeholders for certain elements
// that are handled by SubstituteParameters
var roleParameters = map[common.SystemRole][]string{
	common.SystemRoleArchiver: []string{
		"%CFG%",
		"%IDX%",
		"%SAMPLE_COUNT%",
	},
	common.SystemRoleAtomizerCliWatchtower: []string{
		"%CFG%",
		"%IDX%",
		"%SIGN_TXS%",
		"0",
	},
	common.SystemRoleRaftAtomizer: []string{"%CFG%", "%IDX%"},
	common.SystemRoleSentinel:     []string{"%CFG%", "%IDX%"},
	common.SystemRoleShard:        []string{"%CFG%", "%IDX%"},
	common.SystemRoleCoordinator: []string{
		"%CFG%",
		"%COORDINATORIDX%",
		"%COORDINATORNODEIDX%",
	},
	common.SystemRoleWatchtower: []string{"%CFG%", "%IDX%"},
	common.SystemRoleShardTwoPhase: []string{
		"%CFG%",
		"%SHARDIDX%",
		"%SHARDNODEIDX%",
	},
	common.SystemRoleTwoPhaseGen:      []string{"%CFG%", "%IDX%"},
	common.SystemRoleSentinelTwoPhase: []string{"%CFG%", "%IDX%"},
	common.SystemRoleAgent: []string{"--loglevel=%LOGLEVEL%",
		"--component_id=%IDX%"},
	common.SystemRoleRuntimeLockingShard: []string{
		"--loglevel=%LOGLEVEL%",
		"--component_id=%SHARDIDX%",
		"--node_id=%SHARDNODEIDX%",
	},
	common.SystemRoleTicketMachine: []string{
		"--loglevel=%LOGLEVEL%",
		"--component_id=%IDX%",
	},
	common.SystemRolePhaseTwoGen: []string{
		"--component_id=%IDX%",
		"--loadgen_accounts=%ACCOUNTS%",
		"--loadgen_agent_affinity=%LGAFFINITY%",
	},
}

// StartRoleBinaries is a convenience method to start a set of test run roles
// from a particular test run on the agents that are supposed to run those
// roles. Gets passed the current set of running commands and will return the
// set with the commands run by this routine appended. The method uses
// AgentsManager.ExecuteCommand to do the actual command execution. See the
// documentation for that method for explanation for the parameter `wait`.
// `envs` is the map of agent ID to environment ID, which is created from the
// main test run logic when deploying the binaries. `cmd` is a channel where
// executed commands get signaled to by the ExecuteCommand method.
func (t *TestRunManager) StartRoleBinaries(
	cmds []runningCommand,
	roles []*common.TestRunRole,
	tr *common.TestRun,
	envs map[int32][]byte,
	cmd chan *common.ExecutedCommand,
	wait bool,
) ([]runningCommand, error) {

	// We run each command in parallel using a goroutine, so we need a
	// waitgroup to synchronize these
	var wg sync.WaitGroup
	wg.Add(len(roles))

	// Make a local error to keep track of errors
	errs := make([]error, 0)

	// Create a local mutex to prevent editing the cmds and errs array from
	// multiple goroutines at once
	cmdLock := sync.Mutex{}

	for _, rl := range roles {
		go func(r *common.TestRunRole) {
			// Use SubstituteParameters to replace the placeholders in the
			// roleParameters with the values based on the testrun and role
			params := make([]string, 0)
			params = append(params, tr.Params...)
			params = append(
				params,
				t.SubstituteParameters(roleParameters[r.Role], r, tr)...)

			t.WriteLog(
				tr,
				"Starting %s on agent %d with parameters %v",
				roleBinaries[r.Role],
				r.AgentID,
				params,
			)

			// Instruct the agent to run the actual command, and get the ID
			// under which the command is running.
			cmdID, err := t.am.ExecuteCommand(
				r.AgentID,
				roleBinaries[r.Role],
				params,
				[]string{
					fmt.Sprintf("TESTRUN_ID=%s", tr.ID),
					fmt.Sprintf("TESTRUN_ROLE=%s-%d", r.Role, r.Index),
				},
				envs[r.AgentID],
				"",
				15000,
				cmd,
				wait,
				true,
				tr.RunPerf,
				tr.PerfSampleRate,
				tr.Debug,
				tr.RecordNetworkTraffic,
			)
			cmdLock.Lock()
			if err != nil {
				// If an error occurred, write it to the test run log and
				// append it to the errors array
				t.WriteLog(
					tr,
					"Error occurred starting %s on agent %d: %s",
					roleBinaries[r.Role],
					r.AgentID,
					err,
				)
				errs = append(errs, err)
			} else {
				// The command started succesfully on the agent, so append the
				// command's id associated with the agent ID to the running
				// commands array
				cmds = append([]runningCommand{{
					agentID:   r.AgentID,
					commandID: cmdID,
				}}, cmds...)
			}
			cmdLock.Unlock()

			// Signal the completion of this goroutine
			wg.Done()
		}(rl)
	}

	// Wait for all goroutines to complete - which means all commands were
	// either started or yielded an error
	wg.Wait()
	if len(errs) > 0 {
		// There were errors starting the commands, so we have to abort the
		// test run here. We have written the individual errors to the testrun
		// log in the goroutine above - so we just return the error count as
		// the main error from this method.
		errStr := fmt.Sprintf(
			"%d errors occurred starting the binaries",
			len(errs),
		)
		// Break and terminate any command that already was succesfully started
		stopErr := t.BreakAndTerminateAllCmds(tr, cmds)
		if stopErr != nil {
			errStr = fmt.Sprintf("%s\n%s", errStr, stopErr.Error())
		}
		return []runningCommand{}, errors.New(errStr)
	}

	// Success
	return cmds, nil
}

// FilterCommandsByRole filters the allCmds array by only including commands
// running on agents that have a particular system role
func (t *TestRunManager) FilterCommandsByRole(
	tr *common.TestRun,
	allCmds []runningCommand,
	role common.SystemRole,
) []runningCommand {
	filteredCommands := []runningCommand{}
	for _, r := range tr.Roles {
		if r.Role == role {
			for i, cmd := range allCmds {
				if cmd.agentID == r.AgentID {
					filteredCommands = append(filteredCommands, allCmds[i])
				}
			}
		}
	}
	return filteredCommands
}

// BreakAndTerminateAllCmds will instruct the agent runnning a command to send a
// os.Interrupt signal followed by an os.Kill signal to it, for each of the
// commands in the runningCommands array.
func (t *TestRunManager) BreakAndTerminateAllCmds(
	tr *common.TestRun,
	cmds []runningCommand,
) error {
	err := t.BreakAllCmds(tr, cmds)
	if err != nil {
		return err
	}
	time.Sleep(time.Second * 5)
	return t.TerminateAllCmds(tr, cmds)
}

// TerminateAllCmds will instruct the agent runnning a command to send a
// an os.Kill signal to it, for each of the commands in the runningCommands
// array.
func (t *TestRunManager) TerminateAllCmds(
	tr *common.TestRun,
	cmds []runningCommand,
) error {
	errs := make([]error, 0)
	wg := sync.WaitGroup{}
	wg.Add(len(cmds))
	t.WriteLog(tr, "Terminating %d commands", len(cmds))
	for i := range cmds {
		go func(cmd runningCommand) {
			err := t.am.TerminateCommand(cmd.agentID, cmd.commandID)
			if err != nil && err != coordinator.ErrAgentNotFound {
				t.WriteLog(
					tr,
					"Error terminating command %x on agent %d: %s",
					cmd.commandID,
					cmd.agentID,
					err,
				)
				errs = append(errs, err)
			}
			wg.Done()
		}(cmds[i])
	}
	wg.Wait()

	if len(errs) > 0 {
		return fmt.Errorf("%d error(s) occurred stopping commands", len(errs))
	}
	t.WriteLog(tr, "Terminated %d commands", len(cmds))
	return nil
}

// BreakAllCmds will instruct the agent runnning a command to send a
// os.Interrupt signal, for each of the
// commands in the runningCommands array.
func (t *TestRunManager) BreakAllCmds(
	tr *common.TestRun,
	cmds []runningCommand,
) error {
	errs := make([]error, 0)
	wg := sync.WaitGroup{}
	wg.Add(len(cmds))
	t.WriteLog(tr, "Interrupting %d commands", len(cmds))
	for i := range cmds {
		go func(cmd runningCommand) {
			err := t.am.BreakCommand(cmd.agentID, cmd.commandID)
			if err != nil && err != coordinator.ErrAgentNotFound {
				t.WriteLog(
					tr,
					"Error breaking command %x on agent %d: %s",
					cmd.commandID,
					cmd.agentID,
					err,
				)
				errs = append(errs, err)
			}
			wg.Done()
		}(cmds[i])
	}
	wg.Wait()
	if len(errs) > 0 {
		return fmt.Errorf("%d error(s) occurred breaking commands", len(errs))
	}
	t.WriteLog(tr, "Interrupted %d commands", len(cmds))
	return nil
}

// HandleCommandFailure is called when one of the commands fails during the
// testrun. It will kill all the other commands, download the performance
// profiles and outputs that are available for inspection.
func (t *TestRunManager) HandleCommandFailure(
	tr *common.TestRun,
	allCmds []runningCommand,
	envs map[int32][]byte,
	fail *common.ExecutedCommand,
) error {
	err := t.BreakAndTerminateAllCmds(tr, allCmds)
	if err != nil {
		t.WriteLog(tr, "Error breaking commands: %v", err)
	}

	// Even if commands fail, the performance profiles might be
	// interesting and they are available. So why not?
	time.Sleep(5 * time.Second)
	err = t.GetPerformanceProfiles(tr, allCmds, envs)
	if err != nil {
		t.WriteLog(tr, "Error getting performance profiles: %v", err)
	}

	err = t.GetLogFiles(tr, allCmds, envs)
	if err != nil {
		t.WriteLog(tr, "Error getting log files: %v", err)
	}

	// Instruct all agents to upload their outputs, but ignore it if the files
	// do not exist (yet) because we failed before they were created. We're
	// interested in the files that are available
	err = t.CopyOutputs(tr, envs, true)
	if err != nil {
		t.WriteLog(tr, "Error copying outputs: %v", err)
	}

	t.FailTestRun(
		tr,
		fmt.Errorf(
			"command %s [%s] on agent %d failed with exit code %d",
			fail.CommandID,
			fail.Description,
			fail.AgentID,
			fail.ExitCode,
		),
	)
	return nil
}

// RunBinaries is a convenience method that will execute the correct method
// based on the architecture configured for the test run, or return an error
// if the architecture is unknown
func (t *TestRunManager) RunBinaries(
	tr *common.TestRun,
	envs map[int32][]byte,
	cmd chan *common.ExecutedCommand,
	failures chan *common.ExecutedCommand,
) error {
	if t.IsAtomizer(tr.Architecture) {
		return t.RunBinariesAtomizer(tr, envs, cmd, failures)
	} else if t.Is2PC(tr.Architecture) {
		return t.RunBinariesTwoPhase(tr, envs, cmd, failures)
	} else if t.IsPhaseTwo(tr.Architecture) {
		return t.RunBinariesPhaseTwo(tr, envs, cmd, failures)
	}
	return fmt.Errorf("unknown architecture: [%s]", tr.Architecture)
}
