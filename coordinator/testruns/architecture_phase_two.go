package testruns

import (
	"fmt"
	"strings"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
)

func (t *TestRunManager) IsParsec(architectureID string) bool {
	return strings.HasPrefix(architectureID, "parsec")
}

// RunBinariesParsec will orchestrate the running of all roles for a full
// cycle test with the PArSEC architecture
func (t *TestRunManager) RunBinariesParsec(
	tr *common.TestRun,
	envs map[int32][]byte,
	cmd chan *common.ExecutedCommand,
	failures chan *common.ExecutedCommand,
) error {
	// Build the sequence of commands to start
	startSequence := t.CreateStartSequenceParsec(tr)

	// Execute the sequence of commands to start
	allCmds, terminated, err := t.executeStartSequence(
		tr,
		startSequence,
		envs,
		cmd,
		failures,
	)
	if err != nil {
		cuerr := t.CleanupCommands(tr, allCmds, envs)
		if cuerr != nil {
			return cuerr
		}
		return err
	}
	if terminated { // Terminated yields true if the user aborted the testrun
		return nil
	}
	// allCmds now holds all of the running commands for this test run.

	timeout := time.Duration(tr.SampleCount) * time.Second

	t.UpdateStatus(
		tr,
		common.TestRunStatusRunning,
		fmt.Sprintf(
			"Waiting for manual termination or timeout (%.1f minutes)",
			timeout.Minutes(),
		),
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
	case <-tr.TerminateChan:
	case <-time.After(timeout):
	}

	err = t.CleanupCommandsParsec(tr, allCmds, envs)
	if err != nil {
		return err
	}

	// Save the test run & return nil (success)
	t.PersistTestRun(tr)

	return nil
}

func (t *TestRunManager) CleanupCommandsParsec(
	tr *common.TestRun,
	allCmds []runningCommand,
	envs map[int32][]byte,
) error {

	// More sophisticated shutdown sequence:
	// - sigint all loadgens
	// - sigint the agents
	// - wait for agentdelay
	// - sigkill loadgens
	// - sigkill agents
	// - sigint shard
	// - wait 5 seconds
	// - sigkill shard
	// - sigint ticketmachine
	// - wait 5 seconds
	// - sigkill ticketmachine

	t.WriteLog(tr, "Interrupting all loadgens")
	err := t.BreakAllCmds(
		tr,
		t.FilterCommandsByRole(tr, allCmds, common.SystemRoleParsecGen),
	)
	if err != nil {
		return err
	}

	t.WriteLog(tr, "Interrupting all agents")
	err = t.BreakAllCmds(
		tr,
		t.FilterCommandsByRole(tr, allCmds, common.SystemRoleAgent),
	)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)
	if tr.AgentShutdownDelay > 5 {
		time.Sleep(time.Second * time.Duration(tr.AgentShutdownDelay-5))
	}

	t.WriteLog(tr, "Terminating all loadgens")
	err = t.TerminateAllCmds(
		tr,
		t.FilterCommandsByRole(tr, allCmds, common.SystemRoleParsecGen),
	)
	if err != nil {
		return err
	}

	t.WriteLog(tr, "Terminating all agents")
	err = t.TerminateAllCmds(
		tr,
		t.FilterCommandsByRole(tr, allCmds, common.SystemRoleAgent),
	)
	if err != nil {
		return err
	}

	t.WriteLog(tr, "Interrupting and terminating all shards")
	err = t.BreakAndTerminateAllCmds(
		tr,
		t.FilterCommandsByRole(
			tr,
			allCmds,
			common.SystemRoleRuntimeLockingShard,
		),
	)
	if err != nil {
		return err
	}

	t.WriteLog(tr, "Interrupting and terminating all ticket machines")
	err = t.BreakAndTerminateAllCmds(
		tr,
		t.FilterCommandsByRole(tr, allCmds, common.SystemRoleTicketMachine),
	)
	if err != nil {
		return err
	}

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
	return nil
}

// CreateStartSequenceParsec uses the test run configuration to determine in
// which sequence the agent roles should be started, and returns an array of
// startSequenceEntry elements that are ordered in the sequence in which they
// should be started up.
func (t *TestRunManager) CreateStartSequenceParsec(
	tr *common.TestRun,
) []startSequenceEntry {
	// Determine the start sequence
	startSequence := make([]startSequenceEntry, 0)

	roleStartTimeout := time.Minute * 4

	// Raft clusters now elect a random leader so we start them all at once
	ticketMachines := t.GetAllRolesSorted(tr, common.SystemRoleTicketMachine)

	// Start all nodes at once, wait for each node to have their RAFT port
	// available - and for 1 in {rep factor} to have the actual RPC port
	// available.
	startSequence = append(startSequence, startSequenceEntry{
		roles:   ticketMachines,
		timeout: roleStartTimeout,
		waitForPort: []PortIncrement{
			PortIncrementRaftPort,
			PortIncrementDefaultPort,
		},
		waitForPortCount: []int{
			0,
			len(ticketMachines) / tr.ShardReplicationFactor,
		},
	})

	shards := t.GetAllRolesSorted(tr, common.SystemRoleRuntimeLockingShard)

	// Start the shards
	startSequence = append(startSequence, startSequenceEntry{
		roles:   shards,
		timeout: roleStartTimeout,
		waitForPort: []PortIncrement{
			PortIncrementRaftPort,
			PortIncrementDefaultPort,
		},
		waitForPortCount: []int{0, len(shards) / tr.ShardReplicationFactor},
	})

	// Start the agents
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleAgent),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Start all load generators
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleParsecGen),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{}, // Don't wait for anything - loadgens don't accept incoming
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
	if (shardClusters * tr.ShardReplicationFactor) != len(shards) {
		return nil, fmt.Errorf(
			"number of shards [%d] should be a multiple of replication factor [%d]",
			len(shards),
			tr.ShardReplicationFactor,
		)
	}

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

	ret = append(
		ret,
		fmt.Sprintf("--proxy_count=%d", tr.AgentRPCInstances),
	)

	ret = append(ret, fmt.Sprintf("--loadgen_txtype=%s", tr.LoadGenTxType))

	if tr.Telemetry {
		ret = append(ret, "--telemetry=1")
	}

	if tr.ContentionRate > 0 {
		ret = append(
			ret,
			fmt.Sprintf("--contention_rate=%.2f", tr.ContentionRate),
		)
	}

	return ret, nil
}
