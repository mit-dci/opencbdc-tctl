package common

// This file contains the definitions of the system architectures the test
// controller supports, and the roles that exist within those architectures
// the information in AvailableArchitecures is sent to the frontend to inform
// it of the available architectures and roles to pick from when composing a
// new test run
type SystemRole string

const SystemRoleArchiver SystemRole = "archiver"
const SystemRoleAtomizer SystemRole = "atomizer"
const SystemRoleRaftAtomizer SystemRole = "raft-atomizer"
const SystemRoleShard SystemRole = "shard"
const SystemRoleSentinel SystemRole = "sentinel"
const SystemRoleTxPlayer SystemRole = "tx-player"
const SystemRoleWatchtower SystemRole = "watchtower"
const SystemRoleAtomizerCli SystemRole = "atomizer-cli"
const SystemRoleAtomizerCliWatchtower SystemRole = "atomizer-cli-watchtower"
const SystemRoleShardTwoPhase SystemRole = "shard-2pc"
const SystemRoleSentinelTwoPhase SystemRole = "sentinel-2pc"
const SystemRoleCoordinator SystemRole = "coordinator"
const SystemRoleTwoPhaseGen SystemRole = "twophase-gen"
const SystemRoleAgent SystemRole = "agent"
const SystemRoleRuntimeLockingShard SystemRole = "runtime_locking_shard"
const SystemRoleTicketMachine SystemRole = "ticket_machine"
const SystemRolePhaseTwoGen SystemRole = "phasetwo_bench"

type SystemArchitectureRole struct {
	Role       SystemRole `json:"role"`
	Title      string     `json:"title"`
	ShortTitle string     `json:"shortTitle"`
}

type SystemArchitecture struct {
	ID          string                   `json:"id"`
	Name        string                   `json:"name"`
	Roles       []SystemArchitectureRole `json:"roles"`
	DefaultTest *TestRun                 `json:"defaultTest"`
}

var AvailableArchitectures = []SystemArchitecture{
	{
		ID:   "default",
		Name: "RAFT-Atomizer based",
		Roles: []SystemArchitectureRole{
			{
				Role:       SystemRoleArchiver,
				Title:      "Archiver",
				ShortTitle: "Arch",
			},
			{
				Role:       SystemRoleRaftAtomizer,
				Title:      "Atomizer",
				ShortTitle: "Atmzr",
			},
			{
				Role:       SystemRoleShard,
				Title:      "Shard",
				ShortTitle: "Shard",
			},
			{
				Role:       SystemRoleSentinel,
				Title:      "Sentinel",
				ShortTitle: "Sent",
			},
			{
				Role:       SystemRoleAtomizerCli,
				Title:      "CLI",
				ShortTitle: "CLI",
			},
			{
				Role:       SystemRoleAtomizerCliWatchtower,
				Title:      "Watchtower CLI",
				ShortTitle: "WT-CLI",
			},
			{
				Role:       SystemRoleWatchtower,
				Title:      "Watchtower",
				ShortTitle: "WTow",
			},
		},
		DefaultTest: &TestRun{
			Roles:                    make([]*TestRunRole, 0),
			SweepRoles:               make([]*TestRunRole, 0),
			STXOCacheDepth:           10,
			BatchSize:                100000,
			WindowSize:               100000,
			TargetBlockInterval:      250,
			ElectionTimeoutUpper:     4000,
			ElectionTimeoutLower:     3000,
			Heartbeat:                1000,
			RaftMaxBatch:             100000,
			SnapshotDistance:         0,
			ShardReplicationFactor:   2,
			Architecture:             "default",
			SampleCount:              1215,
			LoadGenOutputCount:       2,
			LoadGenInputCount:        2,
			FixedTxRate:              1,
			PreseedCount:             100000000,
			PreseedShards:            true,
			TrimSamplesAtStart:       5,
			TrimZeroesAtStart:        true,
			TrimZeroesAtEnd:          true,
			AtomizerLogLevel:         "WARN",
			SentinelLogLevel:         "WARN",
			ArchiverLogLevel:         "WARN",
			WatchtowerLogLevel:       "WARN",
			ShardLogLevel:            "WARN",
			Debug:                    false,
			RunPerf:                  false,
			BatchDelay:               0,
			InvalidTxRate:            0,
			SkipCleanUp:              false,
			KeepTimedOutAgents:       false,
			Repeat:                   1,
			RetryOnFailure:           false,
			WatchtowerBlockCacheSize: 100,
			WatchtowerErrorCacheSize: 10000000,
			MaxRetries:               1,
			SentinelAttestations:     1,
			AuditInterval:            5,
		},
	},
	{
		ID:   "default-inmem",
		Name: "RAFT-Atomizer based (In-Memory Shard)",
		Roles: []SystemArchitectureRole{
			{
				Role:       SystemRoleArchiver,
				Title:      "Archiver",
				ShortTitle: "Arch",
			},
			{
				Role:       SystemRoleRaftAtomizer,
				Title:      "Atomizer",
				ShortTitle: "Atmzr",
			},
			{
				Role:       SystemRoleShard,
				Title:      "Shard",
				ShortTitle: "Shard",
			},
			{
				Role:       SystemRoleSentinel,
				Title:      "Sentinel",
				ShortTitle: "Sent",
			},
			{
				Role:       SystemRoleAtomizerCli,
				Title:      "CLI",
				ShortTitle: "CLI",
			},
			{
				Role:       SystemRoleAtomizerCliWatchtower,
				Title:      "Watchtower CLI",
				ShortTitle: "WT-CLI",
			},
			{
				Role:       SystemRoleWatchtower,
				Title:      "Watchtower",
				ShortTitle: "WTow",
			},
		},
		DefaultTest: &TestRun{
			Roles:                    make([]*TestRunRole, 0),
			SweepRoles:               make([]*TestRunRole, 0),
			STXOCacheDepth:           10,
			BatchSize:                100000,
			WindowSize:               100000,
			TargetBlockInterval:      250,
			ElectionTimeoutUpper:     4000,
			ElectionTimeoutLower:     3000,
			Heartbeat:                1000,
			RaftMaxBatch:             100000,
			SnapshotDistance:         0,
			ShardReplicationFactor:   2,
			Architecture:             "default-inmem",
			SampleCount:              1215,
			LoadGenOutputCount:       2,
			LoadGenInputCount:        2,
			FixedTxRate:              1,
			PreseedCount:             100000000,
			PreseedShards:            true,
			TrimSamplesAtStart:       5,
			TrimZeroesAtStart:        true,
			TrimZeroesAtEnd:          true,
			AtomizerLogLevel:         "WARN",
			SentinelLogLevel:         "WARN",
			ArchiverLogLevel:         "WARN",
			WatchtowerLogLevel:       "WARN",
			ShardLogLevel:            "WARN",
			Debug:                    false,
			RunPerf:                  false,
			BatchDelay:               0,
			InvalidTxRate:            0,
			SkipCleanUp:              false,
			KeepTimedOutAgents:       false,
			Repeat:                   1,
			RetryOnFailure:           false,
			WatchtowerBlockCacheSize: 100,
			WatchtowerErrorCacheSize: 10000000,
			MaxRetries:               1,
			SentinelAttestations:     1,
			AuditInterval:            5,
		},
	},
	{
		ID:   "2pc",
		Name: "Two-phase Commit",
		Roles: []SystemArchitectureRole{
			{
				Role:       SystemRoleCoordinator,
				Title:      "Coordinator",
				ShortTitle: "Coord",
			},
			{
				Role:       SystemRoleShardTwoPhase,
				Title:      "Shard",
				ShortTitle: "Shard",
			},
			{
				Role:       SystemRoleSentinelTwoPhase,
				Title:      "Sentinel",
				ShortTitle: "Sent",
			},
			{
				Role:       SystemRoleTwoPhaseGen,
				Title:      "Generator",
				ShortTitle: "Gen",
			},
		},
		DefaultTest: &TestRun{
			Roles:                    make([]*TestRunRole, 0),
			SweepRoles:               make([]*TestRunRole, 0),
			STXOCacheDepth:           0,
			BatchSize:                100000,
			WindowSize:               100000,
			TargetBlockInterval:      250,
			ElectionTimeoutUpper:     1000,
			ElectionTimeoutLower:     500,
			Heartbeat:                250,
			RaftMaxBatch:             100000,
			SnapshotDistance:         0,
			ShardReplicationFactor:   3,
			Architecture:             "2pc",
			SampleCount:              315,
			LoadGenOutputCount:       2,
			LoadGenInputCount:        2,
			FixedTxRate:              1,
			PreseedCount:             100000000,
			PreseedShards:            true,
			TrimSamplesAtStart:       5,
			TrimZeroesAtStart:        true,
			TrimZeroesAtEnd:          true,
			AtomizerLogLevel:         "WARN",
			SentinelLogLevel:         "WARN",
			ArchiverLogLevel:         "WARN",
			WatchtowerLogLevel:       "WARN",
			ShardLogLevel:            "WARN",
			Debug:                    false,
			RunPerf:                  false,
			BatchDelay:               1,
			InvalidTxRate:            0,
			SkipCleanUp:              false,
			KeepTimedOutAgents:       false,
			Repeat:                   1,
			RetryOnFailure:           false,
			WatchtowerBlockCacheSize: 100,
			WatchtowerErrorCacheSize: 10000000,
			MaxRetries:               1,
		},
	},
	{
		ID:   "2pc-inmem",
		Name: "Two-phase Commit (In-Memory Shard)",
		Roles: []SystemArchitectureRole{
			{
				Role:       SystemRoleCoordinator,
				Title:      "Coordinator",
				ShortTitle: "Coord",
			},
			{
				Role:       SystemRoleShardTwoPhase,
				Title:      "Shard",
				ShortTitle: "Shard",
			},
			{
				Role:       SystemRoleSentinelTwoPhase,
				Title:      "Sentinel",
				ShortTitle: "Sent",
			},
			{
				Role:       SystemRoleTwoPhaseGen,
				Title:      "Generator",
				ShortTitle: "Gen",
			},
		},
		DefaultTest: &TestRun{
			Roles:                    make([]*TestRunRole, 0),
			SweepRoles:               make([]*TestRunRole, 0),
			STXOCacheDepth:           0,
			BatchSize:                100000,
			WindowSize:               100000,
			TargetBlockInterval:      250,
			ElectionTimeoutUpper:     1000,
			ElectionTimeoutLower:     500,
			Heartbeat:                250,
			RaftMaxBatch:             100000,
			SnapshotDistance:         0,
			ShardReplicationFactor:   3,
			Architecture:             "2pc-inmem",
			SampleCount:              315,
			LoadGenOutputCount:       2,
			LoadGenInputCount:        2,
			FixedTxRate:              1,
			PreseedCount:             100000000,
			PreseedShards:            true,
			TrimSamplesAtStart:       5,
			TrimZeroesAtStart:        true,
			TrimZeroesAtEnd:          true,
			AtomizerLogLevel:         "WARN",
			SentinelLogLevel:         "WARN",
			ArchiverLogLevel:         "WARN",
			WatchtowerLogLevel:       "WARN",
			ShardLogLevel:            "WARN",
			Debug:                    false,
			RunPerf:                  false,
			BatchDelay:               1,
			InvalidTxRate:            0,
			SkipCleanUp:              false,
			KeepTimedOutAgents:       false,
			Repeat:                   1,
			RetryOnFailure:           false,
			WatchtowerBlockCacheSize: 100,
			WatchtowerErrorCacheSize: 10000000,
			MaxRetries:               1,
		},
	},
	{
		ID:   "phase-two",
		Name: "Phase Two Programmability",
		Roles: []SystemArchitectureRole{
			{
				Role:       SystemRoleAgent,
				Title:      "Agent",
				ShortTitle: "Agent",
			},
			{
				Role:       SystemRoleRuntimeLockingShard,
				Title:      "Shard",
				ShortTitle: "Shard",
			},
			{
				Role:       SystemRoleTicketMachine,
				Title:      "Ticket Machine",
				ShortTitle: "Ticketer",
			},
			{
				Role:       SystemRolePhaseTwoGen,
				Title:      "Generator",
				ShortTitle: "Gen",
			},
		},
		DefaultTest: &TestRun{
			Roles:                    make([]*TestRunRole, 0),
			SweepRoles:               make([]*TestRunRole, 0),
			STXOCacheDepth:           0,
			BatchSize:                100000,
			WindowSize:               100000,
			TargetBlockInterval:      250,
			ElectionTimeoutUpper:     1000,
			ElectionTimeoutLower:     500,
			Heartbeat:                250,
			RaftMaxBatch:             100000,
			SnapshotDistance:         0,
			ShardReplicationFactor:   3,
			Architecture:             "phase-two",
			SampleCount:              315,
			LoadGenOutputCount:       2,
			LoadGenInputCount:        2,
			FixedTxRate:              1,
			PreseedCount:             100000000,
			PreseedShards:            false,
			TrimSamplesAtStart:       5,
			TrimZeroesAtStart:        true,
			TrimZeroesAtEnd:          true,
			AtomizerLogLevel:         "WARN",
			SentinelLogLevel:         "WARN",
			ArchiverLogLevel:         "WARN",
			WatchtowerLogLevel:       "WARN",
			ShardLogLevel:            "WARN",
			Debug:                    false,
			RunPerf:                  false,
			BatchDelay:               1,
			InvalidTxRate:            0,
			SkipCleanUp:              false,
			KeepTimedOutAgents:       false,
			Repeat:                   1,
			RetryOnFailure:           false,
			WatchtowerBlockCacheSize: 100,
			WatchtowerErrorCacheSize: 10000000,
			MaxRetries:               1,
		},
	},
}
