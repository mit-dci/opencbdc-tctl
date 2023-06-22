package testruns

import (
	"sort"

	"github.com/mit-dci/opencbdc-tctl/common"
)

// GetAllRolesSorted extracts all roles of a particular type from the set of
// roles in the testrun, and sorts them by Index
func (t *TestRunManager) GetAllRolesSorted(
	tr *common.TestRun,
	role common.SystemRole,
) []*common.TestRunRole {
	ret := make([]*common.TestRunRole, 0)
	for i := range tr.Roles {
		if tr.Roles[i].Role == role {
			ret = append(ret, tr.Roles[i])
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Index < ret[j].Index
	})

	return ret
}

// NormalizeRole converts variants of certain roles (for instance the two-phase
// commit shard) to one standardized role name - this is specifically used when
// generating the configuration file where both locking_shard and shard require
// configuration prefix "shard". Full translation table:
//
// 2PC Shard -> Shard
// 2PC Sentinel -> Sentinel
func (t *TestRunManager) NormalizeRole(
	role common.SystemRole,
) common.SystemRole {
	if role == common.SystemRoleShardTwoPhase {
		role = common.SystemRoleShard
	} else if role == common.SystemRoleSentinelTwoPhase {
		role = common.SystemRoleSentinel
	} else if role == common.SystemRoleAtomizerCliWatchtower || role == common.SystemRoleParsecGen || role == common.SystemRoleTwoPhaseGen {
		role = common.SystemRoleLoadGen
	}
	return role
}
