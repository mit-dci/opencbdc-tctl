package testruns

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/coordinator"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

// resultCalculation is the struct that's used to queue a particular testrun's
// result calculation
type resultCalculation struct {
	calculateForRun *common.TestRun
	responseChan    chan error
}

// ShouldCalculateResults returns if a result calculation is necessary -
// currently only determined by the absence of test results
func (t *TestRunManager) ShouldCalculateResults(tr *common.TestRun) bool {
	return tr.Result == nil
}

// ParallelResultCalculation dictates how many results can be calculated in
// parallel
const ParallelResultCalculation = 4

// ResultCalculator is the main processor for the resultCalculationChan. It will
// be started `ParallelResultCalculation` times in the background and read from
// the channel to see which result calculations need to be performed. Once a
// calculation request has been read from the channel, it will use the result
// calculation python script to produce the results.
func (t *TestRunManager) ResultCalculator() {
	for job := range t.resultCalculationChan {
		tr := job.calculateForRun
		logging.Debugf("Calculating test run %s results", tr.ID)

		// The result calculation script is expected to be placed next to
		// the main coordinator assembly
		exeDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			if job.responseChan != nil {
				job.responseChan <- err
			}
			continue
		}
		calcScript := filepath.Join(exeDir, "calculate_results.py")

		// Create the `plots` subdirectory of the testrun folder where the
		// time series, latency distribution and throughput distribution plots
		// will be written to by the calculation script
		testRunDir := filepath.Join(
			common.DataDir(),
			fmt.Sprintf("testruns/%s", tr.ID),
		)
		err = os.MkdirAll(filepath.Join(testRunDir, "plots"), 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			logging.Errorf("Error creating plots dir: %v", err)
		}

		// Build the command to execute
		cmd := exec.Command("python3", calcScript)

		// Build the environment variables to use for execution, which we
		// base on the trimming parameters set for the test run
		cmd.Env = os.Environ()
		cmd.Env = append(
			cmd.Env,
			fmt.Sprintf("TRIM_SAMPLES=%d", tr.TrimSamplesAtStart),
		)
		cmd.Env = append(
			cmd.Env,
			fmt.Sprintf("BLOCK_TIME=%d", tr.TargetBlockInterval),
		)
		trimZeroes := 1
		if !tr.TrimZeroesAtStart {
			trimZeroes = 0
		}
		trimZeroesEnd := 1
		if !tr.TrimZeroesAtEnd {
			trimZeroesEnd = 0
		}
		cmd.Env = append(
			cmd.Env,
			fmt.Sprintf("TRIM_ZEROES_START=%d", trimZeroes),
		)
		cmd.Env = append(
			cmd.Env,
			fmt.Sprintf("TRIM_ZEROES_END=%d", trimZeroesEnd),
		)
		cmd.Dir = testRunDir

		// Execute the calculation script
		out, err := cmd.CombinedOutput()
		if err != nil {
			if job.responseChan != nil {
				job.responseChan <- err
			}
			logging.Errorf(
				"Could not calculate result for testrun %s: %v",
				tr.ID,
				err,
			)
			logging.Infof("Result calculation output:\r\n%s", string(out))
			continue
		}
		logging.Infof("Result calculation output:\r\n%s", string(out))

		// The python script wrote the results to the results.json file. We
		// load it into the common.TestRun.Results property by using the
		// LoadTestResult method
		t.LoadTestResult(tr)

		// If ShouldCalculateResults still returns true, this means the
		// calculation failed. Log that.
		if t.ShouldCalculateResults(tr) {
			logging.Errorf(
				"Test results are still empty after loading %s: %v",
				tr.ID,
				tr.Result,
			)
		} else {
			// Notify the real time channel that the result is available - this
			// will trigger the frontend to show the results
			t.ev <- coordinator.Event{
				Type: coordinator.EventTypeTestRunResultAvailable,
				Payload: coordinator.TestRunResultAvailablePayload{
					TestRunID: tr.ID,
					Result:    tr.Result,
				},
			}
		}
		// If there was a response chan, signal the succesful completion of the
		// result calculation
		if job.responseChan != nil {
			job.responseChan <- nil
		}
	}
}

// CalculateResults will enqueue the result calculation if needed onto the job
// channel and await its completion, returning the result. Use `recalc` set to
// `true` to force calculation even if results are already present
func (t *TestRunManager) CalculateResults(
	tr *common.TestRun,
	recalc bool,
) (*common.TestResult, error) {
	if t.ShouldCalculateResults(tr) || recalc {
		rc := make(chan error, 1)
		t.resultCalculationChan <- resultCalculation{
			calculateForRun: tr,
			responseChan:    rc,
		}
		return tr.Result, <-rc
	}
	return tr.Result, nil
}
