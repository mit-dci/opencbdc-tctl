package testruns

import (
	"fmt"
	"strings"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
)

func (t *TestRunManager) IsPhaseTwo(architectureID string) bool {
	return strings.HasPrefix(architectureID, "phase-two")
}

// RunBinariesPhaseTwo will orchestrate the running of all roles for a full
// cycle test with the phase two architecture
func (t *TestRunManager) RunBinariesPhaseTwo(
	tr *common.TestRun,
	envs map[int32][]byte,
	cmd chan *common.ExecutedCommand,
	failures chan *common.ExecutedCommand,
) error {
	gensDone := make(chan []runningCommand, 1)

	// Build the sequence of commands to start
	startSequence := t.CreateStartSequencePhaseTwo(tr, gensDone)

	// Execute the sequence of commands to start
	allCmds, terminated, err := t.executeStartSequence(
		tr,
		startSequence,
		envs,
		cmd,
		failures,
	)
	if err != nil {
		return err
	}
	if terminated { // Terminated yields true if the user aborted the testrun
		return nil
	}
	// allCmds now holds all of the running commands for this test run.

	t.UpdateStatus(
		tr,
		common.TestRunStatusRunning,
		"Waiting for manual termination or timeout (5 minutes)",
	)

	// Run the failure scenario in a separate goroutine. Pass it a channel
	// that will get a true sent to it when we exit the test - such that if
	// the test fails for whatever reason (or is manually terminated) the
	// scheduled failures are no longer executed.
	cancelFailures := make(chan bool, 1)
	go t.FailRoles(tr, cancelFailures)
	defer func() {
		cancelFailures <- true
	}()

	// This section waits for either one of the roles to fail (case 1), or the
	// user to manually terminate the run (case 3), or load gens end
	// successfully
	// (case 2 - success case)
	select {
	case fail := <-failures:
		return t.HandleCommandFailure(tr, allCmds, envs, fail)
	case waitCmds := <-gensDone:
		allCmds = append(allCmds, waitCmds...)
	case <-tr.TerminateChan:
	}

	// Break all commands that are still running - if one command fails or only
	// the archiver has succesfully completed, we still need to terminate all
	// the other commands using a interrupt or kill signal. This would trigger
	// the finishing of all stdout/err buffers and terminating any performance
	// profiling running alongside the commands
	err = t.BreakAndTerminateAllCmds(tr, allCmds)
	if err != nil {
		return err
	}

	// Time for the commands to break and commit perf results
	time.Sleep(5 * time.Second)

	// Trigger the agents to upload the performance data for all commands
	// to S3
	err = t.GetPerformanceProfiles(tr, allCmds, envs)
	if err != nil {
		return err
	}

	err = t.GetLogFiles(tr, allCmds, envs)
	if err != nil {
		return err
	}

	// Save the test run & return nil (success)
	t.PersistTestRun(tr)

	return nil
}

// CreateStartSequencePhaseTwo uses the test run configuration to determine in
// which sequence the agent roles should be started, and returns an array of
// startSequenceEntry elements that are ordered in the sequence in which they
// should be started up.
func (t *TestRunManager) CreateStartSequencePhaseTwo(
	tr *common.TestRun,
	gensDone chan []runningCommand,
) []startSequenceEntry {
	// Determine the start sequence
	startSequence := make([]startSequenceEntry, 0)

	roleStartTimeout := time.Minute * 1

	// Divide the set of shard roles into leaders (node index 0) and followers
	ticketMachines := t.GetAllRolesSorted(tr, common.SystemRoleTicketMachine)
	followerTicketMachines := make([]*common.TestRunRole, 0)
	leaderTicketMachines := make([]*common.TestRunRole, 0)
	for i := 0; i < len(ticketMachines); i++ {
		if i%tr.ShardReplicationFactor == 0 {
			leaderTicketMachines = append(
				leaderTicketMachines,
				ticketMachines[i],
			)
		} else {
			followerTicketMachines = append(followerTicketMachines, ticketMachines[i])
		}
	}

	// First start the follower ticket machines, then the leaders
	startSequence = append(startSequence, startSequenceEntry{
		roles:       followerTicketMachines,
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementRaftPort},
	}, startSequenceEntry{
		roles:       leaderTicketMachines,
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Divide the set of shard roles into leaders (node index 0) and followers
	shards := t.GetAllRolesSorted(tr, common.SystemRoleRuntimeLockingShard)
	followerShards := make([]*common.TestRunRole, 0)
	leaderShards := make([]*common.TestRunRole, 0)
	for i := 0; i < len(shards); i++ {
		if i%tr.ShardReplicationFactor == 0 {
			leaderShards = append(leaderShards, shards[i])
		} else {
			followerShards = append(followerShards, shards[i])
		}
	}

	// Start the shards
	startSequence = append(startSequence, startSequenceEntry{
		roles:       followerShards,
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementRaftPort},
	}, startSequenceEntry{
		roles:       leaderShards,
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Start the agents
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleAgent),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Start all load generators
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRolePhaseTwoGen),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{}, // Don't wait for anything - loadgens don't accept incoming
		doneChan:    gensDone,
	})
	return startSequence
}

func (t *TestRunManager) GenerateParams(tr *common.TestRun) ([]string, error) {
	ret := make([]string, 0)

	ticket_machines := t.GetAllRolesSorted(tr, common.SystemRoleTicketMachine)
	if len(ticket_machines) < 1 {
		return nil, fmt.Errorf("at least one ticket machine is required")
	}

	ret = append(
		ret,
		fmt.Sprintf("--ticket_machine_count=%d", len(ticket_machines)),
	)

	for i, s := range ticket_machines {
		a, err := t.coord.GetAgent(
			s.AgentID,
		)
		if err != nil {
			return nil, err
		}
		ret = append(
			ret,
			fmt.Sprintf(
				"--ticket_machine%d_endpoint=%s:5000",
				i,
				a.SystemInfo.PrivateIPs[0],
			),
		)
	}

	shards := t.GetAllRolesSorted(tr, common.SystemRoleRuntimeLockingShard)
	if len(shards) == 0 {
		return nil, fmt.Errorf("at least one shard is required")
	}

	shardClusters := len(shards) / tr.ShardReplicationFactor

	ret = append(ret, fmt.Sprintf("--shard_count=%d", shardClusters))

	for i := 0; i < shardClusters; i++ {
		ret = append(
			ret,
			fmt.Sprintf("--shard%d_count=%d", i, tr.ShardReplicationFactor),
		)

		for j := 0; j < tr.ShardReplicationFactor; j++ {
			s := shards[(i*tr.ShardReplicationFactor)+j]

			a, err := t.coord.GetAgent(
				s.AgentID,
			)
			if err != nil {
				return nil, err
			}
			ret = append(
				ret,
				fmt.Sprintf(
					"--shard%d%d_endpoint=%s:5000",
					i,
					j,
					a.SystemInfo.PrivateIPs[0],
				),
			)
		}
	}

	agents := t.GetAllRolesSorted(tr, common.SystemRoleAgent)
	if len(agents) == 0 {
		return nil, fmt.Errorf("at least one agent is required")
	}

	ret = append(ret, fmt.Sprintf("--agent_count=%d", len(agents)))

	for i, s := range agents {
		a, err := t.coord.GetAgent(
			s.AgentID,
		)
		if err != nil {
			return nil, err
		}
		ret = append(
			ret,
			fmt.Sprintf(
				"--agent%d_endpoint=%s:5000",
				i,
				a.SystemInfo.PrivateIPs[0],
			),
		)
	}

	return ret, nil
}
