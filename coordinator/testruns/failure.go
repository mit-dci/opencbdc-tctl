package testruns

import (
	"encoding/hex"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
)

// FailRoles is run in a goroutine by RunBinaries to fail roles that were
// configured to fail at a certain point in the test run. The `cancel` channel
// is monitored, if anything is sent there the failure logic is aborted. This is
// mainly the case when the main executing logic fails and aborts the test run.
func (t *TestRunManager) FailRoles(tr *common.TestRun, cancel chan bool) {
	started := time.Now()

	for {
		// Loop over all roles to determine the next failure that we should
		// execute
		var nextFailureRole *common.TestRunRole
		nextFailureRole = nil
		nextFailureTime := time.Now().Add(24 * time.Hour)
		for _, r := range tr.Roles {
			if r.Failure != nil && !r.Failure.Failed {
				failureTime := started.Add(
					time.Second * time.Duration(r.Failure.After),
				)
				if failureTime.Before(nextFailureTime) {
					nextFailureTime = failureTime
					nextFailureRole = r
				}
			}
		}

		// If there is no next failure (there were either none defined or we
		// executed them all), exit this loop
		if nextFailureRole == nil {
			return
		}

		// Write to the log which failure we will execute next
		t.WriteLog(
			tr,
			"Waiting until %s to fail role %s %d (Agent %d)",
			nextFailureTime.String(),
			string(nextFailureRole.Role),
			nextFailureRole.Index,
			nextFailureRole.AgentID,
		)

		// Monitor the cancel channel until it's time to enforce the next
		// failure
		for time.Until(nextFailureTime) > 0 {
			select {
			case <-cancel:
				return
			case <-time.After(1 * time.Second):
			}
		}

		// Execute the defined failure
		t.WriteLog(
			tr,
			"Failing role %s %d (Agent %d)",
			string(nextFailureRole.Role),
			nextFailureRole.Index,
			nextFailureRole.AgentID,
		)

		// Get all running commands on the role we need to fail
		running := t.am.RunningCommandsForAgent(nextFailureRole.AgentID)
		t.WriteLog(
			tr,
			"Killing %d commands on agent %d",
			len(running),
			nextFailureRole.AgentID,
		)
		for _, cmd := range running {
			// Append this command as being failed on purpose - such that the
			// test run is not aborted because of this command's failure.
			tr.DeliberateFailures = append(tr.DeliberateFailures, cmd.CommandID)
			cmdID, _ := hex.DecodeString(cmd.CommandID)
			t.WriteLog(
				tr,
				"Killing command %s (id %x) on agent %d",
				cmd.Command,
				cmdID,
				cmd.AgentID,
			)
			err := t.am.TerminateCommand(cmd.AgentID, cmdID)
			if err != nil {
				t.WriteLog(tr, "Could not kill command %x: %v", cmdID, err)
			}
		}
		// Set the failure as having executed succesfully, so that it's not
		// being considered as the next failure anymore.
		nextFailureRole.Failure.Failed = true

	}
}
