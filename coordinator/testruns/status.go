package testruns

import (
	"fmt"
	"strings"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/coordinator"
)

// UpdateStatus will set the status property of the testrun, append the new
// status to the test run log if it's not a duplicate of the current status.
// It will also set the start/complete time to Now() if those times are Zero and
// the status is Running / Completed. It will also send this status update over
// the real-time event channel such that the UI can update these statuses in
// real-time
func (t *TestRunManager) UpdateStatus(
	tr *common.TestRun,
	newStatus common.TestRunStatus,
	details string,
) {

	if tr.Status == newStatus && tr.Details == details {
		// Why bother?
		return
	}

	// Check if the status is only a percentile increase - to prevent cluttering
	// the log. So if the status updates are "Deploying binaries (1%)" ...
	// "Deploying binaries (2%)" ... etc, we will not log each of them - only
	// the first one
	shouldLog := true
	if strings.Contains(details, "(") && strings.Contains(details, "%") {
		currentStatus := tr.Details
		if strings.Contains(currentStatus, "(") {
			currentStatus = currentStatus[:strings.Index(currentStatus, "(")]
			currentStatus = strings.TrimRight(currentStatus, " ")
		}
		trimmedDetails := details[:strings.Index(details, "(")]
		trimmedDetails = strings.TrimRight(trimmedDetails, " ")
		if trimmedDetails == currentStatus {
			shouldLog = false
		}
	}
	if shouldLog {
		t.WriteLog(
			tr,
			"Updated status to [%s] [%s]",
			string(newStatus),
			details,
		)
	}
	tr.Status = newStatus

	// Set start/complete time if the time is Zero and the status indicates that
	// the testrun is being started/completed
	if newStatus == common.TestRunStatusRunning && tr.Started.IsZero() {
		tr.Started = time.Now()
	}
	if (newStatus == common.TestRunStatusFailed || newStatus == common.TestRunStatusCompleted) &&
		tr.Completed.IsZero() {
		tr.Completed = time.Now()
	}
	if details != "" {
		tr.Details = details
	}

	// Send the update over the realtime channel
	t.ev <- coordinator.Event{
		Type: coordinator.EventTypeTestRunStatusChanged,
		Payload: coordinator.TestRunStatusChangePayload{
			TestRunID: tr.ID,
			Status:    string(tr.Status),
			Started:   tr.Started,
			Completed: tr.Completed,
			Details:   tr.Details,
		},
	}

	// Persist the testrun to ensure the status is preserved
	t.PersistTestRun(tr)
}

// FailTestRun will set the status of a testrun to failed, with the given
// error as reason. It will then terminate the AWS roles that are still active
// and copy any test run outputs/performance data that was uploaded to S3 before
// the test had failed. Lastly, it will reschedule the test run if it was
// configure to be rescheduled on failures
func (t *TestRunManager) FailTestRun(tr *common.TestRun, err error) {
	t.WriteLog(tr, "Test run failed: [%s]", err.Error())

	// If the test run has roles running in AWS, we need to terminate all of
	// them when the test fails.
	if t.HasAWSRoles(tr) {
		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			"Run failed, killing spawned AWS agents",
		)
		err2 := t.KillAwsAgents(tr)
		if err2 != nil {
			t.UpdateStatus(
				tr,
				common.TestRunStatusFailed,
				fmt.Sprintf(
					"Failed (%v), and unable to kill AWS all agent(s): %v",
					err,
					err2,
				),
			)
			return
		}
	}

	// Even for failed runs, we might have interesting performance profiles
	// or partially complete outputs. Since they're available anyway it doesn't
	// hurt to copy them
	if len(tr.PendingResultDownloads) > 0 {
		s3Err := t.awsm.DownloadMultipleFromS3(tr.PendingResultDownloads)
		if s3Err != nil {
			t.WriteLog(tr, "Failed to download outputs from S3: %v", s3Err)
		}
	}

	// Prevent double failures leading to rescheduling twice
	if tr.Status == common.TestRunStatusFailed {
		tr.WriteLog(
			"Another failure recorded after the test run was already failed - this should be avoided",
		)
	} else {
		// Set the status to failed
		t.UpdateStatus(tr, common.TestRunStatusFailed, err.Error())
		// Test runs can be configured to be retried when they fail. This is
		// scheduled here.
		if tr.RetryOnFailure {
			t.Reschedule(tr)
		}
	}
}
