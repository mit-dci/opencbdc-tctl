package common

import (
	"crypto/sha256"
	"encoding/json"
	"math"

	"github.com/mit-dci/opencbdc-tctl/logging"
)

// TestRunNormalizedConfig is a flat representation of a testrun's configuration
// including all parameters that are relevant for comparing test run results
// between one another. When creating matrices of results, test runs that have
// all of these parameters in common will be bucketed into the same result row.
// Using the raw `common.TestRun` type for this doesn't work, since data such as
// agent IDs, AWS instance IDs will differ for every run - hence this normalized
// representation that can be generated from the raw testrun.
type TestRunNormalizedConfig struct {
	Architecture           string  `json:"architectureID"`
	HourUTC                int     `json:"hourUTC"`
	DayUTC                 int     `json:"dayUTC"`
	MonthUTC               int     `json:"monthUTC"`
	AtomizerCount          int     `json:"atomizers"`
	ClientCount            int     `json:"clients"`
	WatchtowerCount        int     `json:"watchtowers"`
	SentinelCount          int     `json:"sentinels"`
	ShardCount             int     `json:"shards"`
	CoordinatorCount       int     `json:"coordinators"`
	BatchSize              int     `json:"batchSize"`
	WindowSize             int     `json:"windowSize"`
	ShardReplicationFactor int     `json:"shardReplicationFactor"`
	STXOCacheDepth         int     `json:"stxoCacheDepth"`
	TargetBlockInterval    int     `json:"targetBlockInterval"`
	ElectionTimeoutUpper   int     `json:"electionTimeoutUpper"`
	ElectionTimeoutLower   int     `json:"electionTimeoutLower"`
	Heartbeat              int     `json:"heartbeat"`
	RaftMaxBatch           int     `json:"raftMaxBatch"`
	SnapshotDistance       int     `json:"snapshotDistance"`
	LoadGenOutputCount     int     `json:"loadGenOutputCount"`
	LoadGenInputCount      int     `json:"loadGenInputCount"`
	InvalidTxRate          float64 `json:"invalidTxRate"`
	FixedTxRate            float64 `json:"fixedTxRate"`
	BatchDelay             int     `json:"batchDelay"`
	ShardCPU               int     `json:"shardCPU"`
	SentinelCPU            int     `json:"sentinelCPU"`
	WatchtowerCPU          int     `json:"watchtowerCPU"`
	AtomizerCPU            int     `json:"atomizerCPU"`
	CoordinatorCPU         int     `json:"coordinatorCPU"`
	ClientCPU              int     `json:"clientCPU"`
	ShardRAM               int     `json:"shardRAM"`
	SentinelRAM            int     `json:"sentinelRAM"`
	WatchtowerRAM          int     `json:"watchtowerRAM"`
	AtomizerRAM            int     `json:"atomizerRAM"`
	CoordinatorRAM         int     `json:"coordinatorRAM"`
	ClientRAM              int     `json:"clientRAM"`
	MultiRegion            bool    `json:"multiRegion"`
	CommitHash             string  `json:"commitHash"`
	ControllerCommitHash   string  `json:"controllerCommitHash"`
	PreseedCount           int64   `json:"preseedCount"`
	PreseedShards          bool    `json:"preseedShards"`
	LoadGenAccounts        int     `json:"loadGenAccounts"`
	ContentionRate         float64 `json:"contentionRate"`
}

// Calculates a hash over the normalized config by hashing the serialized JSON
// representation. Can be used to easily determine if two
// TestRunNormalizedConfig object are equal
func (trc *TestRunNormalizedConfig) Hash() []byte {
	b, err := json.Marshal(trc)
	if err != nil {
		logging.Warnf("Unable to marshal testrun config: %v", err)
	}
	h := sha256.Sum256(b)
	return h[:]
}

// NormalizedConfig generates a TestRunNormalizedConfig from a raw
// common.TestRun including the agent data
func (tr *TestRun) NormalizedConfig() *TestRunNormalizedConfig {
	return tr.NormalizedConfigWithAgentData(true)
}

// NormalizedConfigWithAgentData generates a TestRunNormalizedConfig from a raw
// common.TestRun - where `agentData` determines if it is including the CPU and
// RAM for each of the roles. If this agent data should be omitted from the
// config and set to 0 instead, use `agentData = false`. In this case, two test
// runs with equal numbers of sentinels/shards/etc, but with these roles running
// in different configurations (i.e. one test run runs 2 sentinels on a 2vCPU
// instance where the other runs 2 sentinels on a 8vCPU instance) they would
// still have the same TestRunNormalizedConfig
func (tr *TestRun) NormalizedConfigWithAgentData(
	agentData bool,
) *TestRunNormalizedConfig {
	var trc TestRunNormalizedConfig

	// First simply use JSON marshal and unmarshal to set all of the
	// easily copyable properties of the config - note that their JSON field
	// names are equal between the common.TestRun and
	// common.TestRunNormalizedConfig structs
	b, err := json.Marshal(tr)
	if err != nil {
		logging.Warnf("Unable to marshal testrun: %v", err)
	}
	err = json.Unmarshal(b, &trc)
	if err != nil {
		logging.Warnf("Unable to unmarshal testrun: %v", err)
	}

	// Normalize snapshot distance. 0 = disabled, previously we used
	// a massive distance to force disabling.
	if trc.SnapshotDistance == 1000000000 {
		trc.SnapshotDistance = 0
	}

	// Calculate the count and average CPU and RAM for each role type
	// Also identify in which region all these roles run - used to determine
	// if we run multi-region.
	cpu := map[SystemRole]float64{}
	ram := map[SystemRole]float64{}
	count := map[SystemRole]int{}
	regions := map[string]bool{}
	for _, r := range tr.Roles {
		_, ok := count[r.Role]
		if !ok {
			cpu[r.Role] = 0
			ram[r.Role] = 0
			count[r.Role] = 0
		}
		count[r.Role] = count[r.Role] + 1

		if agentData {
			agentSysInfo := AgentSystemInfo{}
			agentRegion := ""
			for _, ad := range tr.AgentDataAtStart {
				if ad.AgentID == r.AgentID {
					agentSysInfo = ad.SystemInfo
					agentRegion = ad.AwsRegion
				}
			}
			cpu[r.Role] = cpu[r.Role] + float64(agentSysInfo.NumCPU)
			ram[r.Role] = ram[r.Role] + float64(agentSysInfo.TotalMemory)
			regions[agentRegion] = true
		}
	}

	if agentData {
		for k := range cpu {
			cpu[k] = cpu[k] / float64(count[k])
		}
		for k := range ram {
			ram[k] = ram[k] / float64(count[k])
		}
	}

	// Prevent float rounding from causing mismatch
	trc.InvalidTxRate = math.Round(trc.InvalidTxRate*100) / 100
	trc.FixedTxRate = math.Round(trc.FixedTxRate*100) / 100

	// Set the properties for CPU/RAM for each role type
	trc.AtomizerCPU = int(
		getFloatValueForAnyKey(cpu, SystemRoleRaftAtomizer),
	)
	trc.ClientCPU = int(
		getFloatValueForAnyKey(
			cpu,
			SystemRoleAtomizerCliWatchtower,
			SystemRoleTwoPhaseGen,
			SystemRoleParsecGen,
		),
	)
	trc.SentinelCPU = int(
		getFloatValueForAnyKey(
			cpu,
			SystemRoleSentinel,
			SystemRoleSentinelTwoPhase,
		),
	)
	trc.ShardCPU = int(
		getFloatValueForAnyKey(
			cpu,
			SystemRoleShard,
			SystemRoleShardTwoPhase,
			SystemRoleRuntimeLockingShard,
		),
	)
	trc.WatchtowerCPU = int(getFloatValueForAnyKey(cpu, SystemRoleWatchtower))
	trc.CoordinatorCPU = int(
		getFloatValueForAnyKey(cpu, SystemRoleCoordinator, SystemRoleAgent),
	)
	trc.AtomizerRAM = int(
		getFloatValueForAnyKey(
			ram,
			SystemRoleRaftAtomizer,
		) / 1024,
	)
	trc.ClientRAM = int(
		getFloatValueForAnyKey(
			ram,
			SystemRoleAtomizerCliWatchtower,
			SystemRoleTwoPhaseGen,
			SystemRoleParsecGen,
		) / 1024,
	)
	trc.SentinelRAM = int(
		getFloatValueForAnyKey(
			ram,
			SystemRoleSentinel,
			SystemRoleSentinelTwoPhase,
		) / 1024,
	)
	trc.ShardRAM = int(
		getFloatValueForAnyKey(
			ram,
			SystemRoleShard,
			SystemRoleShardTwoPhase,
			SystemRoleRuntimeLockingShard,
		) / 1024,
	)
	trc.WatchtowerRAM = int(
		getFloatValueForAnyKey(ram, SystemRoleWatchtower) / 1024,
	)
	trc.CoordinatorRAM = int(
		getFloatValueForAnyKey(
			ram,
			SystemRoleCoordinator,
			SystemRoleAgent,
		) / 1024,
	)

	// Set the role counts
	trc.AtomizerCount = getIntValueForAnyKey(
		count,
		SystemRoleRaftAtomizer,
	)
	trc.ClientCount = getIntValueForAnyKey(
		count,
		SystemRoleAtomizerCliWatchtower,
		SystemRoleTwoPhaseGen,
		SystemRoleParsecGen,
	)
	trc.SentinelCount = getIntValueForAnyKey(
		count,
		SystemRoleSentinel,
		SystemRoleSentinelTwoPhase,
	)
	trc.ShardCount = getIntValueForAnyKey(
		count,
		SystemRoleShard,
		SystemRoleShardTwoPhase,
		SystemRoleRuntimeLockingShard,
	)
	trc.WatchtowerCount = getIntValueForAnyKey(count, SystemRoleWatchtower)
	trc.CoordinatorCount = getIntValueForAnyKey(
		count,
		SystemRoleCoordinator,
		SystemRoleAgent,
	)

	// Set the multiregion property
	trc.MultiRegion = len(regions) > 1

	// For time sweeps, we want to bucket the test runs by hour/day/month - for
	// other run types we don't want to involve started time since it would end
	// up splitting our buckets into more than what we want ( we want to bucket
	// test runs in different hours / days to remain in the same bucket )
	if tr.Sweep == "time" {
		trc.HourUTC = tr.Started.UTC().Hour()
		trc.DayUTC = tr.Started.UTC().Day()
		trc.MonthUTC = int(tr.Started.UTC().Month())
	}
	return &trc
}

func getFloatValueForAnyKey(
	m map[SystemRole]float64,
	keys ...SystemRole,
) float64 {
	for _, k := range keys {
		_, ok := m[k]
		if ok {
			return m[k]
		}
	}
	return 0
}

func getIntValueForAnyKey(m map[SystemRole]int, keys ...SystemRole) int {
	for _, k := range keys {
		_, ok := m[k]
		if ok {
			return m[k]
		}
	}
	return 0
}
