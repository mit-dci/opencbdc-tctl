package testruns

import (
	"fmt"

	"github.com/mit-dci/opencbdc-tct/common"
)

// executeStartSequence executes a pregenerated sequence of commands to start
// up
func (t *TestRunManager) executeStartSequence(
	tr *common.TestRun,
	startSequence []startSequenceEntry,
	envs map[int32][]byte,
	cmd chan *common.ExecutedCommand,
	failures chan *common.ExecutedCommand,
) ([]runningCommand, bool, error) {
	allCmds := []runningCommand{}
	var err error

	// Now that we have the full start sequence, execute it
	for _, seq := range startSequence {
		if len(seq.roles) == 0 {
			continue
		}
		// Show the starting status in the frontend
		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			fmt.Sprintf("Starting %d %s(s)", len(seq.roles), seq.roles[0].Role),
		)

		// Launch the command(s) and append them to the allCmds array
		if seq.doneChan != nil {
			go func(done chan []runningCommand, errChan chan error) {
				newCmds, err := t.StartRoleBinaries(
					[]runningCommand{},
					seq.roles,
					tr,
					envs,
					cmd,
					true,
				)
				if err != nil {
					errChan <- err
				}
				// FIXME: return newCmds in the error case as well
				done <- newCmds
			}(seq.doneChan, seq.errChan)
		} else {
			allCmds, err = t.StartRoleBinaries(allCmds, seq.roles, tr, envs, cmd, false)
			if err != nil {
				return allCmds, false, err
			}
		}

		// If needed, wait for the component to respond to a TCP port connection
		if len(seq.waitForPort) > 0 {
			for _, p := range seq.waitForPort {
				err = t.WaitForRolesOnline(tr, seq.roles, p, seq.timeout)
				if err != nil {
					return allCmds, false, err
				}
			}
		}

		// This checks if the user has preemtively terminated the test run and
		// if so, we abort further executing the test
		if terminated := t.TerminateIfNeeded(tr, allCmds, envs, failures); terminated {
			return allCmds, true, nil
		}

		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			fmt.Sprintf("Started %d %s(s)", len(seq.roles), seq.roles[0].Role),
		)
	}
	return allCmds, false, nil
}
