package testruns

import (
	"fmt"
	"strings"

	"github.com/mit-dci/opencbdc-tctl/common"
)

// seed_privkey is used by the loadgens for spending the seeded outputs
var seed_privkey = "a0f36553548b3a66c003413140d7b59e43464ca11af66f25a6e746be501596b7"

// SubstituteParameters will replace placeholders in commands, command line
// parameters, with values based on the role's configuration or index in a
// cluster
func (t *TestRunManager) SubstituteParameters(
	params []string,
	r *common.TestRunRole,
	tr *common.TestRun,
) []string {
	newParams := make([]string, 0)

	for _, p := range params {
		p = strings.ReplaceAll(p, "%CFG%", "config.cfg")
		p = strings.ReplaceAll(p, "%IDX%", fmt.Sprintf("%d", r.Index))
		p = strings.ReplaceAll(
			p,
			"%SHARDIDX%",
			fmt.Sprintf("%d", r.Index/tr.ShardReplicationFactor),
		)
		p = strings.ReplaceAll(
			p,
			"%SHARDNODEIDX%",
			fmt.Sprintf("%d", r.Index%tr.ShardReplicationFactor),
		)
		p = strings.ReplaceAll(
			p,
			"%COORDINATORIDX%",
			fmt.Sprintf("%d", r.Index/tr.ShardReplicationFactor),
		)
		p = strings.ReplaceAll(
			p,
			"%COORDINATORNODEIDX%",
			fmt.Sprintf("%d", r.Index%tr.ShardReplicationFactor),
		)
		p = strings.ReplaceAll(
			p,
			"%SAMPLE_COUNT%",
			fmt.Sprintf("%d", tr.SampleCount),
		)
		p = strings.ReplaceAll(p, "%SIGN_TXS%", "1") // Always sign
		p = strings.ReplaceAll(
			p,
			"%ACCOUNTS%",
			fmt.Sprintf("%d", tr.LoadGenAccounts),
		)
		p = strings.ReplaceAll(p, "%LOGLEVEL%", t.RoleLogLevel(tr, r))
		if r.Role == common.SystemRoleParsecGen {
			p = strings.ReplaceAll(p, "%LGAFFINITY%", t.LoadGenAffinity(tr, r))
		}
		newParams = append(newParams, p)
	}
	return newParams
}

func (t *TestRunManager) LoadGenAffinity(
	tr *common.TestRun,
	r *common.TestRunRole,
) string {
	if !tr.LoadGenAffinity {
		return ""
	}
	affinity := ""
	region := t.awsm.GetLaunchTemplateRegion(r.AwsLaunchTemplateID)
	for _, rr := range tr.Roles {

		if rr.Role == common.SystemRoleAgent {
			roleRegion := t.awsm.GetLaunchTemplateRegion(rr.AwsLaunchTemplateID)
			if roleRegion == region {
				if affinity != "" {
					affinity += ","
				}
				affinity += fmt.Sprintf("%d", rr.Index)
			}
		}
	}
	return affinity
}
