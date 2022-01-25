package testruns

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mit-dci/cbdc-test-controller/common"
)

// Is2PC returns wheter the given architectureID is a two-phase commit
// architecture - this stems from the time when 2PC existed in both an on-disk
// and in-memory shard configuration
func (t *TestRunManager) Is2PC(architectureID string) bool {
	return strings.HasPrefix(architectureID, "2pc")
}

// RunBinariesTwoPhase will orchestrate the running of all roles for a full
// cycle test with the two-phase commit architecture
func (t *TestRunManager) RunBinariesTwoPhase(
	tr *common.TestRun,
	envs map[int32][]byte,
	cmd chan *common.ExecutedCommand,
	failures chan *common.ExecutedCommand,
) error {

	// Build the sequence of commands to start
	startSequence := t.CreateStartSequenceTwoPhase(tr)
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
	// user to manually terminate the run (case 2), or five minutes elapsing
	// (case 3 - success case)
	select {
	case fail := <-failures:
		return t.HandleCommandFailure(tr, allCmds, envs, fail)
	case <-tr.TerminateChan:
	case <-time.After(5 * time.Minute):
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

// GenerateConfigTwoPhase creates a configuration file to place on all nodes
// such that the system roles can properly find each other and are configured
// as was dictacted by the scheduled test definition in the UI
func (t *TestRunManager) GenerateConfigTwoPhase(
	tr *common.TestRun,
) ([]byte, error) {
	var err error
	// The cfg buffer will hold the configuration file's contents
	// after calling all of the below sub methods for generation
	var cfg bytes.Buffer

	if err = t.writeShardConfigTwoPhase(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writeCoordinatorConfigTwoPhase(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writeEndpointConfig(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writeLogLevelConfig(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writeTestRunConfigVariables(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writePreseedConfigVariables(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writeRoleCounts(&cfg, tr); err != nil {
		return nil, err
	}
	if _, err = cfg.Write([]byte("2pc=1\n")); err != nil {
		return nil, err
	}
	if err = t.writeSentinelKeys(&cfg, tr); err != nil {
		return nil, err
	}
	return cfg.Bytes(), nil
}

// writeCoordinatorConfigTwoPhase writes the section of the configuration file
// for the two-phase commit coordinator(s)
func (t *TestRunManager) writeCoordinatorConfigTwoPhase(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	coordinators := t.GetAllRolesSorted(tr, common.SystemRoleCoordinator)
	if len(coordinators) == 0 {
		return errors.New("the system cannot run without a coordinator")
	}

	coordinatorClusters := len(coordinators) / tr.ShardReplicationFactor
	if (coordinatorClusters * tr.ShardReplicationFactor) != len(coordinators) {
		return fmt.Errorf(
			"number of coordinators [%d] should be a multiple of replication factor [%d]",
			len(coordinators),
			tr.ShardReplicationFactor,
		)
	}

	if _, err := cfg.Write(
		[]byte(fmt.Sprintf("coordinator_count=%d\n", coordinatorClusters)),
	); err != nil {
		return err
	}
	coordinatorPortNum := portNums[common.SystemRoleCoordinator]
	for i := 0; i < coordinatorClusters; i++ {
		if _, err := cfg.Write(
			[]byte(
				fmt.Sprintf(
					"coordinator%d_count=%d\n",
					i,
					tr.ShardReplicationFactor,
				),
			),
		); err != nil {
			return err
		}

		for j := 0; j < tr.ShardReplicationFactor; j++ {
			a, err := t.coord.GetAgent(
				coordinators[j+(i*tr.ShardReplicationFactor)].AgentID,
			)
			if err != nil {
				return err
			}
			if _, err := cfg.Write(
				[]byte(
					fmt.Sprintf(
						"coordinator%d_%d_endpoint=\"%s:%d\"\n",
						i,
						j,
						a.SystemInfo.PrivateIPs[0],
						coordinatorPortNum,
					),
				),
			); err != nil {
				return err
			}
			if _, err := cfg.Write(
				[]byte(
					fmt.Sprintf(
						"coordinator%d_%d_raft_endpoint=\"%s:%d\"\n",
						i,
						j,
						a.SystemInfo.PrivateIPs[0],
						coordinatorPortNum+int(PortIncrementRaftPort),
					),
				),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeShardConfigTwoPhase writes the section of the configuration file for the
// two-phase commit shards
func (t *TestRunManager) writeShardConfigTwoPhase(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	shards := t.GetAllRolesSorted(tr, common.SystemRoleShardTwoPhase)
	if len(shards) == 0 {
		return errors.New("the system cannot run without a shard")
	}

	shardClusters := len(shards) / tr.ShardReplicationFactor
	if (shardClusters * tr.ShardReplicationFactor) != len(shards) {
		return fmt.Errorf(
			"number of shards [%d] should be a multiple of replication factor [%d]",
			len(shards),
			tr.ShardReplicationFactor,
		)
	}

	// Write the number of shard clusters to the config file
	if _, err := cfg.Write([]byte(fmt.Sprintf("shard_count=%d\n", shardClusters))); err != nil {
		return err
	}
	shardRange := 256 / shardClusters
	shardPortNum := portNums[common.SystemRoleShardTwoPhase]
	for i := 0; i < shardClusters; i++ {
		// Write the number of nodes in this shard cluster to the config file
		if _, err := cfg.Write(
			[]byte(
				fmt.Sprintf("shard%d_count=%d\n", i, tr.ShardReplicationFactor),
			),
		); err != nil {
			return err
		}

		// Determine the prefix range for this shardcluster
		// and write it to the configuration file
		start := 0 + i*shardRange
		end := ((i + 1) * shardRange) - 1
		if i == shardClusters-1 {
			end = 255
		}
		if _, err := cfg.Write([]byte(fmt.Sprintf("shard%d_start=%d\n", i, start))); err != nil {
			return err
		}
		if _, err := cfg.Write([]byte(fmt.Sprintf("shard%d_end=%d\n", i, end))); err != nil {
			return err
		}

		// Write the endpoints for all the nodes in this shard cluster to the
		// config file
		for j := 0; j < tr.ShardReplicationFactor; j++ {
			a, err := t.coord.GetAgent(
				shards[j+(i*tr.ShardReplicationFactor)].AgentID,
			)
			if err != nil {
				return err
			}
			if _, err := cfg.Write(
				[]byte(
					fmt.Sprintf(
						"shard%d_%d_endpoint=\"%s:%d\"\n",
						i,
						j,
						a.SystemInfo.PrivateIPs[0],
						shardPortNum,
					),
				),
			); err != nil {
				return err
			}
			if _, err := cfg.Write(
				[]byte(
					fmt.Sprintf(
						"shard%d_%d_raft_endpoint=\"%s:%d\"\n",
						i,
						j,
						a.SystemInfo.PrivateIPs[0],
						shardPortNum+int(PortIncrementRaftPort),
					),
				),
			); err != nil {
				return err
			}
			if _, err := cfg.Write(
				[]byte(
					fmt.Sprintf(
						"shard%d_%d_readonly_endpoint=\"%s:%d\"\n",
						i,
						j,
						a.SystemInfo.PrivateIPs[0],
						shardPortNum+int(PortIncrementClientPort),
					),
				),
			); err != nil {
				return err
			}
			if _, err := cfg.Write([]byte(fmt.Sprintf("shard%d_audit_log=\"shard%d_audit_log\"\n", i, i))); err != nil {
				return err
			}
		}
	}
	return nil
}

// CreateStartSequenceTwoPhase uses the test run configuration to determine in
// which sequence the agent roles should be started, and returns an array of
// startSequenceEntry elements that are ordered in the sequence in which they
// should be started up.
func (t *TestRunManager) CreateStartSequenceTwoPhase(
	tr *common.TestRun,
) []startSequenceEntry {
	// Determine the start sequence
	startSequence := make([]startSequenceEntry, 0)

	roleStartTimeout := time.Minute * 1
	// Shard timeout is dependent on preseeding, large preseeds can take a while
	// to load into RAM
	shardTimeout := roleStartTimeout
	if tr.PreseedShards {
		shardTimeout = time.Minute * 5
	}

	// Divide the set of shard roles into leaders (node index 0) and followers
	shards := t.GetAllRolesSorted(tr, common.SystemRoleShardTwoPhase)
	followerShardNodes := make([]*common.TestRunRole, 0)
	leaderShardNodes := make([]*common.TestRunRole, 0)
	for i := 0; i < len(shards); i++ {
		if i%tr.ShardReplicationFactor == 0 {
			leaderShardNodes = append(leaderShardNodes, shards[i])
		} else {
			followerShardNodes = append(followerShardNodes, shards[i])
		}
	}

	// First start the follower shards, then the leaders
	startSequence = append(startSequence, startSequenceEntry{
		roles:       followerShardNodes,
		timeout:     shardTimeout,
		waitForPort: []PortIncrement{PortIncrementRaftPort},
	}, startSequenceEntry{
		roles:       leaderShardNodes,
		timeout:     shardTimeout,
		waitForPort: []PortIncrement{PortIncrementClientPort},
	})

	// Divide the set of coordinator roles into leaders (node index 0) and
	// followers
	coordinators := t.GetAllRolesSorted(tr, common.SystemRoleCoordinator)
	followerCoordinatorNodes := make([]*common.TestRunRole, 0)
	leaderCoordinatorNodes := make([]*common.TestRunRole, 0)

	for i := 0; i < len(coordinators); i++ {
		if i%tr.ShardReplicationFactor == 0 {
			leaderCoordinatorNodes = append(
				leaderCoordinatorNodes,
				coordinators[i],
			)
		} else {
			followerCoordinatorNodes = append(followerCoordinatorNodes, coordinators[i])
		}
	}

	// First start the follower coordinators, then the leaders
	startSequence = append(startSequence, startSequenceEntry{
		roles:       followerCoordinatorNodes,
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementRaftPort},
	}, startSequenceEntry{
		roles:       leaderCoordinatorNodes,
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Start all sentinels
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleSentinelTwoPhase),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Start all load generators
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleTwoPhaseGen),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{}, // Don't wait for anything - loadgens don't accept incoming
	})
	return startSequence
}
