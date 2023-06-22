package common

// This file contains the definitions of the system architectures the test
// controller supports, and the roles that exist within those architectures
// the information in AvailableArchitecures is sent to the frontend to inform
// it of the available architectures and roles to pick from when composing a
// new test run
type SystemRole string

const SystemRoleArchiver SystemRole = "archiver"
const SystemRoleRaftAtomizer SystemRole = "atomizer"
const SystemRoleShard SystemRole = "shard"
const SystemRoleSentinel SystemRole = "sentinel"
const SystemRoleWatchtower SystemRole = "watchtower"
const SystemRoleAtomizerCliWatchtower SystemRole = "atomizer-cli-watchtower"
const SystemRoleShardTwoPhase SystemRole = "shard-2pc"
const SystemRoleSentinelTwoPhase SystemRole = "sentinel-2pc"
const SystemRoleCoordinator SystemRole = "coordinator"
const SystemRoleTwoPhaseGen SystemRole = "twophase-gen"
const SystemRoleAgent SystemRole = "agent"
const SystemRoleRuntimeLockingShard SystemRole = "runtime_locking_shard"
const SystemRoleTicketMachine SystemRole = "ticket_machine"
const SystemRoleParsecGen SystemRole = "parsec_bench"
const SystemRoleLoadGen SystemRole = "loadgen"

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
			Roles: []*TestRunRole{
				{Role: SystemRoleArchiver, Index: 0, AgentID: -1},
				{Role: SystemRoleRaftAtomizer, Index: 0, AgentID: -1},
				{Role: SystemRoleShard, Index: 0, AgentID: -1},
				{Role: SystemRoleShard, Index: 1, AgentID: -1},
				{Role: SystemRoleSentinel, Index: 0, AgentID: -1},
				{Role: SystemRoleWatchtower, Index: 0, AgentID: -1},
				{Role: SystemRoleAtomizerCliWatchtower, Index: 0, AgentID: -1},
			},
			SweepRoles:                make([]*TestRunRole, 0),
			STXOCacheDepth:            10,
			BatchSize:                 100000,
			WindowSize:                100000,
			TargetBlockInterval:       250,
			ElectionTimeoutUpper:      4000,
			ElectionTimeoutLower:      3000,
			Heartbeat:                 1000,
			RaftMaxBatch:              100000,
			SnapshotDistance:          0,
			ShardReplicationFactor:    2,
			Architecture:              "default",
			SampleCount:               1215,
			LoadGenOutputCount:        2,
			LoadGenInputCount:         2,
			FixedTxRate:               1,
			PreseedCount:              1000000,
			PreseedShards:             true,
			TrimSamplesAtStart:        5,
			TrimZeroesAtStart:         true,
			TrimZeroesAtEnd:           true,
			AtomizerLogLevel:          "WARN",
			SentinelLogLevel:          "WARN",
			ArchiverLogLevel:          "WARN",
			WatchtowerLogLevel:        "WARN",
			ShardLogLevel:             "WARN",
			CoordinatorLogLevel:       "WARN",
			AgentLogLevel:             "WARN",
			TicketerLogLevel:          "WARN",
			AtomizerTelemetryLevel:    "OFF",
			SentinelTelemetryLevel:    "OFF",
			ArchiverTelemetryLevel:    "OFF",
			WatchtowerTelemetryLevel:  "OFF",
			ShardTelemetryLevel:       "OFF",
			CoordinatorTelemetryLevel: "OFF",
			AgentTelemetryLevel:       "OFF",
			TicketerTelemetryLevel:    "OFF",
			Debug:                     false,
			RunPerf:                   false,
			BatchDelay:                0,
			InvalidTxRate:             0,
			SkipCleanUp:               false,
			KeepTimedOutAgents:        false,
			Repeat:                    1,
			RetryOnFailure:            false,
			WatchtowerBlockCacheSize:  100,
			WatchtowerErrorCacheSize:  10000000,
			MaxRetries:                1,
			SentinelAttestations:      1,
			AuditInterval:             400,
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
			Roles: []*TestRunRole{
				{Role: SystemRoleCoordinator, Index: 0, AgentID: -1},
				{Role: SystemRoleCoordinator, Index: 1, AgentID: -1},
				{Role: SystemRoleCoordinator, Index: 2, AgentID: -1},
				{Role: SystemRoleShardTwoPhase, Index: 0, AgentID: -1},
				{Role: SystemRoleShardTwoPhase, Index: 1, AgentID: -1},
				{Role: SystemRoleShardTwoPhase, Index: 2, AgentID: -1},
				{Role: SystemRoleSentinelTwoPhase, Index: 0, AgentID: -1},
				{Role: SystemRoleTwoPhaseGen, Index: 0, AgentID: -1},
			},
			SweepRoles:                make([]*TestRunRole, 0),
			STXOCacheDepth:            0,
			BatchSize:                 100000,
			WindowSize:                100000,
			TargetBlockInterval:       250,
			ElectionTimeoutUpper:      1000,
			ElectionTimeoutLower:      500,
			Heartbeat:                 250,
			RaftMaxBatch:              100000,
			SnapshotDistance:          0,
			ShardReplicationFactor:    3,
			Architecture:              "2pc",
			SampleCount:               315,
			LoadGenOutputCount:        2,
			LoadGenInputCount:         2,
			FixedTxRate:               1,
			PreseedCount:              1000000,
			PreseedShards:             true,
			TrimSamplesAtStart:        5,
			TrimZeroesAtStart:         true,
			TrimZeroesAtEnd:           true,
			AtomizerLogLevel:          "WARN",
			SentinelLogLevel:          "WARN",
			ArchiverLogLevel:          "WARN",
			WatchtowerLogLevel:        "WARN",
			ShardLogLevel:             "WARN",
			CoordinatorLogLevel:       "WARN",
			AgentLogLevel:             "WARN",
			TicketerLogLevel:          "WARN",
			AtomizerTelemetryLevel:    "OFF",
			SentinelTelemetryLevel:    "OFF",
			ArchiverTelemetryLevel:    "OFF",
			WatchtowerTelemetryLevel:  "OFF",
			ShardTelemetryLevel:       "OFF",
			CoordinatorTelemetryLevel: "OFF",
			AgentTelemetryLevel:       "OFF",
			TicketerTelemetryLevel:    "OFF",
			Debug:                     false,
			RunPerf:                   false,
			BatchDelay:                1,
			InvalidTxRate:             0,
			SkipCleanUp:               false,
			KeepTimedOutAgents:        false,
			Repeat:                    1,
			RetryOnFailure:            false,
			WatchtowerBlockCacheSize:  100,
			WatchtowerErrorCacheSize:  10000000,
			MaxRetries:                1,
			SentinelAttestations:      1,
			AuditInterval:             60,
		},
	},
	{
		ID:   "parsec",
		Name: "PArSEC",
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
				Role:       SystemRoleParsecGen,
				Title:      "Generator",
				ShortTitle: "Gen",
			},
		},
		DefaultTest: &TestRun{
			// TODO for Phase 2
			Roles:                     make([]*TestRunRole, 0),
			SweepRoles:                make([]*TestRunRole, 0),
			STXOCacheDepth:            0,
			BatchSize:                 100000,
			WindowSize:                100000,
			TargetBlockInterval:       250,
			ElectionTimeoutUpper:      1000,
			ElectionTimeoutLower:      500,
			Heartbeat:                 250,
			RaftMaxBatch:              100000,
			SnapshotDistance:          0,
			ShardReplicationFactor:    3,
			Architecture:              "parsec",
			SampleCount:               315,
			LoadGenOutputCount:        2,
			LoadGenInputCount:         2,
			LoadGenAccounts:           100,
			LoadGenTxType:             "transfer",
			FixedTxRate:               1,
			PreseedCount:              100000000,
			PreseedShards:             false,
			TrimSamplesAtStart:        5,
			TrimZeroesAtStart:         true,
			TrimZeroesAtEnd:           true,
			AtomizerLogLevel:          "WARN",
			SentinelLogLevel:          "WARN",
			ArchiverLogLevel:          "WARN",
			WatchtowerLogLevel:        "WARN",
			ShardLogLevel:             "WARN",
			CoordinatorLogLevel:       "WARN",
			AgentLogLevel:             "WARN",
			TicketerLogLevel:          "WARN",
			AtomizerTelemetryLevel:    "OFF",
			SentinelTelemetryLevel:    "OFF",
			ArchiverTelemetryLevel:    "OFF",
			WatchtowerTelemetryLevel:  "OFF",
			ShardTelemetryLevel:       "OFF",
			CoordinatorTelemetryLevel: "OFF",
			AgentTelemetryLevel:       "OFF",
			TicketerTelemetryLevel:    "OFF",
			LoadGenTelemetryLevel:     "OFF",
			Debug:                     false,
			RunPerf:                   false,
			BatchDelay:                1,
			InvalidTxRate:             0,
			SkipCleanUp:               false,
			KeepTimedOutAgents:        false,
			Repeat:                    1,
			RetryOnFailure:            false,
			WatchtowerBlockCacheSize:  100,
			WatchtowerErrorCacheSize:  10000000,
			MaxRetries:                1,
			SentinelAttestations:      1,
			AuditInterval:             60,
			ContentionRate:            0.0,
		},
	},
}

func ConfigureCommitForDefaultTests(hash string) {
	for i := 0; i < len(AvailableArchitectures); i++ {
		AvailableArchitectures[i].DefaultTest.CommitHash = hash
	}
}

// Sets the appropriate template IDs for the default test based on two types
// the small machine (2vcpu) and the big machine (8vcpu)
func ConfigureLaunchTemplatesForDefaultTests(
	templateIDSmall, templateIDLarge string,
) {
	largeRoles := []SystemRole{
		SystemRoleRaftAtomizer,
		SystemRoleShard,
		SystemRoleShardTwoPhase,
		SystemRoleCoordinator,
		SystemRoleWatchtower,
		SystemRoleSentinelTwoPhase,
	}

	for i := 0; i < len(AvailableArchitectures); i++ {
		for j := 0; j < len(AvailableArchitectures[i].DefaultTest.Roles); j++ {
			large := false
			for _, r := range largeRoles {
				if r == AvailableArchitectures[i].DefaultTest.Roles[j].Role {
					large = true
				}
			}
			if large {
				AvailableArchitectures[i].DefaultTest.Roles[j].AwsLaunchTemplateID = templateIDLarge
			} else {
				AvailableArchitectures[i].DefaultTest.Roles[j].AwsLaunchTemplateID = templateIDSmall
			}
		}
	}

}
