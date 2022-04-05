package testruns

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

// CalculateFlameGraph uses a python script to turn the data gathered by running
// commands with `perf` enabled into a flame graph. This uses the code from
// https://github.com/brendangregg/FlameGraph, which is cloned next to the
// coordinator binary by the Dockerfile
func (t *TestRunManager) CalculateFlameGraph(
	tr *common.TestRun,
	commandID string,
) error {
	t.testRunResultsLock.Lock()
	defer t.testRunResultsLock.Unlock()
	logging.Infof("Calculating flame graph for command %s", commandID)

	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	cmd := exec.Command("bash", "calculate_flamegraph.sh", tr.ID, commandID)
	cmd.Dir = exeDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Warnf(
			"Unable to calculate flame graph: %v\n\n%s",
			err,
			string(out),
		)
		return err
	}

	return nil
}
