package testruns

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
)

func (t *TestRunManager) GenerateConfig(tr *common.TestRun) ([]byte, error) {
	t.UpdateStatus(tr, common.TestRunStatusRunning, "Generating config")
	if t.Is2PC(tr.Architecture) {
		return t.GenerateConfigTwoPhase(tr)
	} else if t.IsAtomizer(tr.Architecture) {
		return t.GenerateConfigAtomizer(tr)
	} else {
		params, err := t.GenerateParams(tr)
		if err != nil {
			return nil, err
		}
		tr.Params = params
		var cfg bytes.Buffer
		for _, param := range params {
			_, err := cfg.WriteString(fmt.Sprintf("%s\n", param))
			if err != nil {
				return nil, err
			}
		}
		return cfg.Bytes(), nil
	}
}

// writeTestRunConfigVariables writes generic testrun-level configuration
// variables such as target block time, batch delay to the config file
func (t *TestRunManager) writeTestRunConfigVariables(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	if _, err := cfg.Write([]byte(fmt.Sprintf("stxo_cache_depth=%d\n", tr.STXOCacheDepth))); err != nil {
		return err
	}
	if _, err := cfg.Write([]byte(fmt.Sprintf("window_size=%d\n", tr.WindowSize))); err != nil {
		return err
	}
	if _, err := cfg.Write([]byte(fmt.Sprintf("batch_size=%d\n", tr.BatchSize))); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf(
				"watchtower_block_cache_size=%d\n",
				tr.WatchtowerBlockCacheSize,
			),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf(
				"watchtower_error_cache_size=%d\n",
				tr.WatchtowerErrorCacheSize,
			),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf(
				"audit_interval=%d\n",
				tr.AuditInterval,
			),
		),
	); err != nil {
		return err
	}

	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf(
				"attestation_threshold=%d\n",
				tr.SentinelAttestations,
			),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf("target_block_interval=%d\n", tr.TargetBlockInterval),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf("election_timeout_upper=%d\n", tr.ElectionTimeoutUpper),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf("election_timeout_lower=%d\n", tr.ElectionTimeoutLower),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write([]byte(fmt.Sprintf("heartbeat=%d\n", tr.Heartbeat))); err != nil {
		return err
	}
	if _, err := cfg.Write([]byte(fmt.Sprintf("raft_max_batch=%d\n", tr.RaftMaxBatch))); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(fmt.Sprintf("snapshot_distance=%d\n", tr.SnapshotDistance)),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf(
				"loadgen_sendtx_input_count=%d\n",
				tr.LoadGenInputCount,
			),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(
			fmt.Sprintf(
				"loadgen_sendtx_output_count=%d\n",
				tr.LoadGenOutputCount,
			),
		),
	); err != nil {
		return err
	}
	if _, err := cfg.Write(
		[]byte(fmt.Sprintf("loadgen_invalid_tx_rate=%f\n", tr.InvalidTxRate)),
	); err != nil {
		return err
	}
	if _, err := cfg.Write([]byte(fmt.Sprintf("loadgen_fixed_tx_rate=%f\n", tr.FixedTxRate))); err != nil {
		return err
	}
	if _, err := cfg.Write([]byte(fmt.Sprintf("batch_delay=%d\n", tr.BatchDelay))); err != nil {
		return err
	}
	return nil
}

// writePreseedConfigVariables writes configuration variables connected to the
// preseeding of the system
func (t *TestRunManager) writePreseedConfigVariables(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	if tr.PreseedShards {
		if _, err := cfg.Write(
			[]byte(
				"\nseed_privkey=\"" + seed_privkey + "\"\n",
			),
		); err != nil {
			return err
		}
		if _, err := cfg.Write([]byte("seed_value=1000000\n")); err != nil {
			return err
		}
		if _, err := cfg.Write([]byte("seed_from=0\n")); err != nil {
			return err
		}
		if _, err := cfg.Write([]byte(fmt.Sprintf("seed_to=%d\n", tr.PreseedCount))); err != nil {
			return err
		}
	}
	return nil
}

func (t *TestRunManager) countRoles(
	tr *common.TestRun,
) map[common.SystemRole]int {
	num := map[common.SystemRole]int{}

	for _, r := range tr.Roles {
		// Increase the role count in the number map
		numPre, ok := num[r.Role]
		if ok {
			num[r.Role] = numPre + 1
		} else {
			num[r.Role] = 1
		}
	}
	return num
}

func (t *TestRunManager) writeRoleCounts(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	num := t.countRoles(tr)
	// Write role counts tallied in the loop above so that the system knows
	// how many of each role exist.
	loadgens := t.GetAllRolesSorted(tr, common.SystemRoleTwoPhaseGen)
	loadgens = append(
		loadgens,
		t.GetAllRolesSorted(tr, common.SystemRoleAtomizerCliWatchtower)...)
	if _, err := cfg.Write([]byte(fmt.Sprintf("loadgen_count=%d\n", len(loadgens)))); err != nil {
		return err
	}
	for k, v := range num {
		_, ok := portNums[k]
		if ok {
			if k == common.SystemRoleShardTwoPhase ||
				k == common.SystemRoleCoordinator {
				// Already done in the separate methods for shards/coordinators
				continue
			}
			role := t.NormalizeRole(k)
			if _, err := cfg.Write([]byte(fmt.Sprintf("%s_count=%d\n", role, v))); err != nil {
				return err
			}
		}
	}
	return nil
}

// RoleLogLevel determines the configured log level for this role, defaulting to
// WARN
func (t *TestRunManager) RoleLogLevel(
	tr *common.TestRun,
	r *common.TestRunRole,
) string {
	loglevel := "WARN"
	switch r.Role {
	case common.SystemRoleRaftAtomizer:
		loglevel = tr.AtomizerLogLevel
	case common.SystemRoleShard:
		fallthrough
	case common.SystemRoleShardTwoPhase:
		fallthrough
	case common.SystemRoleRuntimeLockingShard:
		loglevel = tr.ShardLogLevel
	case common.SystemRoleTicketMachine:
		loglevel = tr.TicketerLogLevel
	case common.SystemRoleAgent:
		loglevel = tr.AgentLogLevel
	case common.SystemRoleSentinel:
		fallthrough
	case common.SystemRoleSentinelTwoPhase:
		loglevel = tr.SentinelLogLevel
	case common.SystemRoleArchiver:
		loglevel = tr.ArchiverLogLevel
	case common.SystemRoleWatchtower:
		loglevel = tr.WatchtowerLogLevel
	case common.SystemRoleCoordinator:
		loglevel = tr.CoordinatorLogLevel
	}
	return loglevel
}

// writeLogLevelConfig writes the role-level error logging level to the config
// file for all roles in the testrun
func (t *TestRunManager) writeLogLevelConfig(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	for _, r := range tr.Roles {
		loglevel := t.RoleLogLevel(tr, r)
		if _, err := cfg.Write(
			[]byte(
				fmt.Sprintf(
					"%s%d_loglevel=\"%s\"\n",
					string(t.NormalizeRole(r.Role)),
					r.Index,
					loglevel,
				),
			),
		); err != nil {
			return err
		}
	}
	return nil
}

// writeEndpointConfig writes role-level endpoint configuration variables
// to the config file, except for
func (t *TestRunManager) writeEndpointConfig(
	cfg io.Writer,
	tr *common.TestRun,
) error {
	for _, r := range tr.Roles {
		// Get the agent on which this role is supposed to run
		a, err := t.coord.GetAgent(r.AgentID)
		if err != nil {
			return err
		}

		// Use the agent (IP) data and the role's regular port endpoint from the
		// portNums map to generate the endpoint at which the role is supposed
		// to listen, and write it to the configuration
		portNum, ok := portNums[r.Role]
		if ok {
			if r.Role == common.SystemRoleShardTwoPhase ||
				r.Role == common.SystemRoleCoordinator {
				// Endpoints already written in the separate configuration for
				// shard and coordinator clusters
				continue
			}
			// Default suffix
			suffix := "_endpoint"

			// For the watchtower the "normal" endpoint through which the
			// system components communicate is called the _internal_endpoint
			if r.Role == common.SystemRoleWatchtower {
				suffix = "_internal_endpoint"
			}
			if _, err := cfg.Write(
				[]byte(
					fmt.Sprintf(
						"%s%d%s=\"%s:%d\"\n",
						string(t.NormalizeRole(r.Role)),
						r.Index,
						suffix,
						a.SystemInfo.PrivateIPs[0],
						portNum,
					),
				),
			); err != nil {
				return err
			}

			// RAFT Atomizers also have a RAFT endpoint, define it with
			// an offset against the normal port number
			if r.Role == common.SystemRoleRaftAtomizer {
				if _, err := cfg.Write(
					[]byte(
						fmt.Sprintf(
							"%s%d_raft_endpoint=\"%s:%d\"\n",
							t.NormalizeRole(r.Role),
							r.Index,
							a.SystemInfo.PrivateIPs[0],
							portNum+int(PortIncrementRaftPort),
						),
					),
				); err != nil {
					return err
				}
			}

			// Atomizer watchtowers also have a client endpoint, define it with
			// an offset against the normal port number
			if r.Role == common.SystemRoleWatchtower {
				if _, err := cfg.Write(
					[]byte(
						fmt.Sprintf(
							"%s%d_client_endpoint=\"%s:%d\"\n",
							t.NormalizeRole(r.Role),
							r.Index,
							a.SystemInfo.PrivateIPs[0],
							portNum+int(PortIncrementClientPort),
						),
					),
				); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// startSequenceEntry describes a individual entry of a (set of) role(s) to be
// started, how long to wait for the role to be started and which port offset
// to wait for to be available
type startSequenceEntry struct {
	roles       []*common.TestRunRole
	timeout     time.Duration
	waitForPort []PortIncrement
	doneChan    chan []runningCommand
	errChan     chan error
}
