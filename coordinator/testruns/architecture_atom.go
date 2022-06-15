package testruns

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/mit-dci/opencbdc-tctl/common"
)

func (t *TestRunManager) IsAtomizer(architectureID string) bool {
	return strings.HasPrefix(architectureID, "default")
}

func (t *TestRunManager) RunBinariesAtomizer(
	tr *common.TestRun,
	envs map[int32][]byte,
	cmd chan *common.ExecutedCommand,
	failures chan *common.ExecutedCommand,
) error {
	archiverDone := make(chan []runningCommand, 1)
	errChan := make(chan error, 100)

	// Build the sequence of commands to start
	startSequence := t.CreateStartSequenceAtomizer(tr, archiverDone, errChan)
	// Execute the sequence of commands to start
	allCmds, terminated, err := t.executeStartSequence(
		tr,
		startSequence,
		envs,
		cmd,
		failures,
	)

	if terminated { // Terminated yields true if the user aborted the testrun
		return nil
	}
	if err != nil {
		cuerr := t.CleanupCommands(tr, allCmds, envs)
		if cuerr != nil {
			return cuerr
		}
		return err
	}
	// allCmds now holds all of the running commands for this test run.

	t.UpdateStatus(tr, common.TestRunStatusRunning, "System running")

	// This starts the failure scenario execution in a subroutine. This logic
	// will terminate the agents as defined in the failure settings of the
	// test run. If the system run fails for whatever reason, the defer
	// statement, executed when this method exits, will ensure a boolean is sent
	// to the cancelFailures channel, which will make the FailRoles() subroutine
	// exit further execution
	cancelFailures := make(chan bool, 1)
	defer func() {
		cancelFailures <- true
	}()
	go t.FailRoles(tr, cancelFailures)

	// Now wait for any of these three ocurrences: (1 - happy case) the archiver
	// completed after five minutes, which concludes the test. (2) a failure
	// happened in one of the roles causing the test run to be aborted (received
	// through the failures channel), or (3) the user terminates the test
	// manually from the frontend (received through tr.TerminateChan)
	t.UpdateStatus(
		tr,
		common.TestRunStatusRunning,
		"Waiting for archiver to complete",
	)
	select {
	case fail := <-failures:
		return t.HandleCommandFailure(tr, allCmds, envs, fail)
	case waitCmds := <-archiverDone:
		allCmds = append(allCmds, waitCmds...)
	case <-tr.TerminateChan:
	}

	err = t.CleanupCommands(tr, allCmds, envs)
	if err != nil {
		return err
	}

	// Read any errors from the parallelly executed commands (we ran the
	// archiver
	// in a separate goroutine to monitor its completion specifically while
	// continuing the main routine to start all the other commands). If any err
	// ocurred with that, it's written to the error channel which we read here
	// and return the first error we see. In a successful run, this would not
	// yield any error
	close(errChan)
	for e := range errChan {
		return e
	}

	// Persist the test run to disk
	t.PersistTestRun(tr)
	return nil
}

func (t *TestRunManager) GenerateConfigAtomizer(
	tr *common.TestRun,
) ([]byte, error) {
	var cfg bytes.Buffer
	var err error

	if err = t.writeShardConfigAtomizer(&cfg, tr); err != nil {
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
	if err = t.writePreseedConfigVariables(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writeSentinelKeys(&cfg, tr); err != nil {
		return nil, err
	}
	if err = t.writeArchiverConfig(&cfg, tr); err != nil {
		return nil, err
	}

	return cfg.Bytes(), nil
}

// writeSentinelKeys adds a private and public key for each sentinel to the
// config file to be used for signing attestations
func (t *TestRunManager) writeSentinelKeys(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	var sents []*common.TestRunRole
	if t.IsAtomizer(tr.Architecture) {
		sents = t.GetAllRolesSorted(tr, common.SystemRoleSentinel)
	} else if t.Is2PC(tr.Architecture) {
		sents = t.GetAllRolesSorted(tr, common.SystemRoleSentinelTwoPhase)
	}
	for i := range sents {
		privKey := make([]byte, 32)
		_, err := rand.Read(privKey)
		if err != nil {
			return err
		}

		if _, err := cfg.Write([]byte(fmt.Sprintf("sentinel%d_private_key=\"%x\"\n", i, privKey))); err != nil {
			return err
		}
		_, pub := btcec.PrivKeyFromBytes(btcec.S256(), privKey)
		if _, err := cfg.Write([]byte(fmt.Sprintf("sentinel%d_public_key=\"%x\"\n", i, pub.X.Bytes()))); err != nil {
			return err
		}

	}
	return nil
}

// writeArchiverConfig adds a database folder name for each archiver to the
// config file
func (t *TestRunManager) writeArchiverConfig(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	archivers := t.GetAllRolesSorted(tr, common.SystemRoleArchiver)
	for i := range archivers {
		if _, err := cfg.Write([]byte(fmt.Sprintf("archiver%d_db=\"archiver%d\"\n", i, i))); err != nil {
			return err
		}
	}
	return nil
}

// writeShardConfigAtomizer writes the shard configuration for the atomizer
// shards to the config file. This specifically checks if the number of physical
// shards is a multiple of the shard replication factor, and writes the prefix
// ranges for each of the shards to the config file.
func (t *TestRunManager) writeShardConfigAtomizer(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	shards := t.GetAllRolesSorted(tr, common.SystemRoleShard)
	if len(shards) == 0 {
		return errors.New("the system cannot run without shards")
	}

	shardClusters := len(shards) / tr.ShardReplicationFactor

	// Determine the prefix range for each of the shards
	// and write it to the configuration file
	shardRange := 256 / shardClusters
	for _, r := range shards {
		start := 0 + (r.Index%shardClusters)*shardRange
		end := (((r.Index % shardClusters) + 1) * shardRange) - 1
		if (r.Index % shardClusters) == shardClusters-1 {
			end = 255
		}
		if _, err := cfg.Write([]byte(fmt.Sprintf("shard%d_start=%d\n", r.Index, start))); err != nil {
			return err
		}
		if _, err := cfg.Write([]byte(fmt.Sprintf("shard%d_end=%d\n", r.Index, end))); err != nil {
			return err
		}
		if _, err := cfg.Write([]byte(fmt.Sprintf("shard%d_db=\"db\"\n", r.Index))); err != nil {
			return err
		}
		if _, err := cfg.Write([]byte(fmt.Sprintf("shard%d_audit_log=\"shard%d_audit_log\"\n", r.Index, r.Index))); err != nil {
			return err
		}

	}
	return nil
}

// CreateStartSequenceAtomizer uses the test run configuration to determine in
// which sequence the agent roles should be started, and returns an array of
// startSequenceEntry elements that are ordered in the sequence in which they
// should be started up.
func (t *TestRunManager) CreateStartSequenceAtomizer(
	tr *common.TestRun,
	archiverDone chan []runningCommand,
	errChan chan error,
) []startSequenceEntry {
	// Determine the start sequence
	startSequence := make([]startSequenceEntry, 0)

	roleStartTimeout := time.Minute * 1
	// Shard timeout is dependent on preseeding, large preseeds can take a while
	// to load into RAM
	// commit 0688510eef7a16669786a8c6510d82b9a99a93cd rebased
	shardTimeout := roleStartTimeout * 15
	if tr.PreseedShards {
		shardTimeout = time.Minute * 15
	}

	// First start the watchtowers
	startSequence = append(startSequence, startSequenceEntry{
		roles:   t.GetAllRolesSorted(tr, common.SystemRoleWatchtower),
		timeout: roleStartTimeout,
		waitForPort: []PortIncrement{
			PortIncrementDefaultPort,
			PortIncrementClientPort,
		},
	})

	// Next, start the atomizers - all at once. Raft elects a random leader
	// now, so we start all and wait for all RAFT ports to respond, but
	// wait for online 1 to respond to actual rpc
	startSequence = append(startSequence, startSequenceEntry{
		roles: t.GetAllRolesSorted(
			tr,
			common.SystemRoleRaftAtomizer,
		),
		timeout: roleStartTimeout,
		waitForPort: []PortIncrement{
			PortIncrementRaftPort,
			PortIncrementDefaultPort,
		},
		waitForPortCount: []int{0, 1},
	})

	// Next, start the archivers
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleArchiver),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
		doneChan:    archiverDone,
	})

	// Start all shards
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleShard),
		timeout:     shardTimeout, // Use the separate shard timeout
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Start all sentinels
	startSequence = append(startSequence, startSequenceEntry{
		roles:       t.GetAllRolesSorted(tr, common.SystemRoleSentinel),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{PortIncrementDefaultPort},
	})

	// Start all load generators
	startSequence = append(startSequence, startSequenceEntry{
		roles: t.GetAllRolesSorted(
			tr,
			common.SystemRoleAtomizerCliWatchtower,
		),
		timeout:     roleStartTimeout,
		waitForPort: []PortIncrement{}, // Don't wait for anything - loadgens don't accept incoming
	})

	return startSequence
}

// ValidateTestRunAtomizer validates the role composition of the test run for an
// atomizer commit system. Reports all errors back as an array
func (t *TestRunManager) ValidateTestRunAtomizer(
	tr *common.TestRun,
) []error {
	errs := make([]error, 0)

	archivers := t.GetAllRolesSorted(tr, common.SystemRoleArchiver)
	shards := t.GetAllRolesSorted(tr, common.SystemRoleShard)
	sentinels := t.GetAllRolesSorted(tr, common.SystemRoleSentinel)
	watchtowers := t.GetAllRolesSorted(tr, common.SystemRoleWatchtower)
	loadgens := t.GetAllRolesSorted(tr, common.SystemRoleAtomizerCliWatchtower)
	atomizers := t.GetAllRolesSorted(tr, common.SystemRoleRaftAtomizer)

	if len(archivers) < 1 {
		errs = append(errs, errors.New("the system needs at least 1 archiver"))
	}

	if len(shards) < 1 {
		errs = append(errs, errors.New("the system needs at least 1 shard"))
	}

	if len(sentinels) < 1 {
		errs = append(errs, errors.New("the system needs at least 1 sentinel"))
	}

	if len(watchtowers) < 1 {
		errs = append(
			errs,
			errors.New("the system needs at least 1 watchtower"),
		)
	}

	if len(loadgens) < 1 {
		errs = append(
			errs,
			errors.New("the system needs at least 1 load generator"),
		)
	}

	if len(atomizers) < 1 {
		errs = append(errs, errors.New("the system needs at least 1 atomizer"))
	}

	if len(shards)%tr.ShardReplicationFactor != 0 {
		errs = append(errs, fmt.Errorf(
			"number of shards [%d] should be a multiple of replication factor [%d]",
			len(shards),
			tr.ShardReplicationFactor,
		))
	}

	return errs
}
