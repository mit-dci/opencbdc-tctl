package testruns

import (
	"fmt"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
)

// ShouldTerminate does a non-blocking read on the TerminateChan of the given
// testrun and returns true if anything is read from it - signaling that the
// user has manually terminated the run
func (t *TestRunManager) ShouldTerminate(tr *common.TestRun) bool {
	select {
	case <-tr.TerminateChan:
		return true
	case <-time.After(time.Millisecond * 10):
	}
	return false
}

// TerminateIfNeeded will call ShouldTerminate to determine if the test run
// needs to be terminated, or checks the failures channel for any failed command
// that warrants terminating the test run. It then proceeds to takes care of all
// the actions needed to cleanly terminate the test run. Will return true if the
// test run was terminated
func (t *TestRunManager) TerminateIfNeeded(
	tr *common.TestRun,
	allCmds []runningCommand,
	envs map[int32][]byte,
	failures chan *common.ExecutedCommand,
) bool {
	if t.ShouldTerminate(tr) {
		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			"Aborted by user request, killing all commands",
		)
		err := t.BreakAndTerminateAllCmds(tr, allCmds)
		if err != nil {
			// No need to return it, we're going to abort the testrun any way
			logging.Warnf("Error terminating commands: %v", err)
		}

		time.Sleep(5 * time.Second)

		// Upon manual termination, need to kill AWS agents
		if t.HasAWSRoles(tr) {
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				"Run aborted, killing spawned AWS agents",
			)
			err2 := t.KillAwsAgents(tr)
			if err2 != nil {
				// No need to return it, we're going to abort the testrun any
				// way
				// Cleanup will be retried by the check loop in Scheduler()
				logging.Warnf("Error killing AWS agents: %v", err2)
			}
		}

		t.UpdateStatus(
			tr,
			common.TestRunStatusAborted,
			"Aborted by user request",
		)

		return true
	}

	if failures != nil {
		select {
		case fail := <-failures:
			err := t.HandleCommandFailure(tr, allCmds, envs, fail)
			if err != nil {
				logging.Errorf("Error handling command failure: %v", err)
			}
			t.FailTestRun(
				tr,
				fmt.Errorf(
					"command %s [%s] on agent %d failed with exit code %d",
					fail.CommandID,
					fail.Description,
					fail.AgentID,
					fail.ExitCode,
				),
			)
			return true
		default:
		}
	}

	return false
}

// Terminate will terminate a test run. If the test run is queued, it will
// change its status to Canceled. If the testrun is running, it will signal the
// request for termination through the testruns TerminateChan, which is read by
// ShouldTerminate at certain points in the test runs execution logic, at which
// time the test run logic will be terminated cleanly.
func (t *TestRunManager) Terminate(id string) {
	for _, tr := range t.testRuns {
		if tr.ID == id {
			if tr.Status == common.TestRunStatusRunning {
				select {
				case tr.TerminateChan <- true:
				case <-time.After(time.Second * 1):
					//timeout
				}
			}
			if tr.Status == common.TestRunStatusQueued {
				t.UpdateStatus(
					tr,
					common.TestRunStatusCanceled,
					"Canceled by user",
				)
			}
		}
	}
}
