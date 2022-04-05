package testruns

import (
	"github.com/mit-dci/opencbdc-tctl/common"
)

// ValidateTestRun validates the role composition of the test run by calling
// the architecture-specific function and return all errors reported
func (t *TestRunManager) ValidateTestRun(
	tr *common.TestRun,
) []error {

	ret := []error{}
	t.UpdateStatus(tr, common.TestRunStatusRunning, "Validating test run")
	if t.Is2PC(tr.Architecture) {
		ret = t.ValidateTestRunTwoPhase(tr)
	} else if t.IsAtomizer(tr.Architecture) {
		ret = t.ValidateTestRunAtomizer(tr)
	}
	return ret
}
