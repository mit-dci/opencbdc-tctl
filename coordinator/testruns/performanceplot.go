package testruns

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
)

// CalculatePerformancePlot uses a python script to turn the data gathered by
// the performance counters on the agent into a performance metric plot. The
// performance data is stored for each command separately, and this method is
// given the ID of the command to calculate the plot for. plotType can be any of
// the following:
// 		system_memory
// 		network_buffers
// 		cpu_usage
// 		num_threads
// 		process_cpu_usage
// 		process_disk_usage
// 		disk_usage
//      flamegraph
func (t *TestRunManager) CalculatePerformancePlot(
	tr *common.TestRun,
	commandID, plotType string,
) error {
	if plotType == "flamegraph" {
		return t.CalculateFlameGraph(tr, commandID)
	}
	t.testRunResultsLock.Lock()
	defer t.testRunResultsLock.Unlock()
	logging.Infof(
		"Calculating performance plot %s for command %s",
		plotType,
		commandID,
	)
	exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return err
	}
	calcScript := filepath.Join(exeDir, "calculate_performance_metrics.py")
	testRunDir := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s", tr.ID),
	)
	err = os.MkdirAll(filepath.Join(testRunDir, "plots"), 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	cmd := exec.Command("python3", calcScript, commandID, plotType)
	cmd.Dir = testRunDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Warnf(
			"Unable to calculate performance data: %v\n\n%s",
			err,
			string(out),
		)
		return err
	}

	return nil
}
