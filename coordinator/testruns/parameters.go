package testruns

import (
	"fmt"
	"strings"

	"github.com/mit-dci/cbdc-test-controller/common"
)

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
		p = strings.ReplaceAll(p, "%THREADS%", fmt.Sprintf("%d", tr.LoadGenThreads))

		newParams = append(newParams, p)
	}
	return newParams
}
