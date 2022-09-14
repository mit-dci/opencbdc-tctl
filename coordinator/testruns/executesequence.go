package testruns

import (
	"fmt"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
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
		if seq.waitBefore > time.Millisecond {
			t.WriteLog(tr, "Waiting for %.2f seconds before starting %d %s(s)", seq.waitBefore.Seconds(), len(seq.roles), seq.roles[0].Role)
			time.Sleep(seq.waitBefore)
		}
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
			// Default to all for all given wait ports in case waitForPortCount
			// is not specified
			for len(seq.waitForPortCount) < len(seq.waitForPort) {
				seq.waitForPortCount = append(seq.waitForPortCount, 0)
			}
			for i, p := range seq.waitForPort {
				err = t.WaitForRolesOnline(
					tr,
					seq.roles,
					p,
					seq.timeout,
					seq.waitForPortCount[i],
				)
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

// startSequenceEntry describes a individual entry of a (set of) role(s) to be
// started, how long to wait for the role to be started and which port offset
// to wait for to be available
type startSequenceEntry struct {
	roles []*common.TestRunRole
	// waitBefore is an extra option to wait for a fixed duration before
	// executing this entry in the start sequence
	waitBefore time.Duration
	timeout    time.Duration
	// waitForPort has a collection of port increments to test on the roles. It
	// will contact the endpoint where that port increment is supposed to be
	// listening to check if it's online. You can specify multiple which will
	// be tried in sequence they're in the array
	waitForPort []PortIncrement
	// waitForPortCount indicates how many endpoints are expected to respond.
	// If this is zero, we will use len(roles) - i.e. expect all of them to.
	waitForPortCount []int
	doneChan         chan []runningCommand
	errChan          chan error
}
