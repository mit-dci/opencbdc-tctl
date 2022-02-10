package testruns

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/coordinator"
	"github.com/mit-dci/opencbdc-tct/coordinator/awsmgr"
	"github.com/mit-dci/opencbdc-tct/logging"
)

// ScheduleTestRun will add the given testrun to the set of queued testruns.
// This method will assign a testrun its ID and set the creation time, initiate
// certain fields with their defaults if not set, persist it and broadcast it
// over the real-time channel so that other users connected to the system will
// learn about the test run's existence without refreshing the browser
func (t *TestRunManager) ScheduleTestRun(tr *common.TestRun) {
	t.testRunsLock.Lock()
	defer t.testRunsLock.Unlock()
	var err error
	tr.ID, err = common.RandomID(12)
	if err != nil {
		logging.Errorf("Error getting randomness user: %s", err.Error())
		return
	}

	tr.Created = time.Now()
	tr.Completed = time.Date(0001, 1, 1, 00, 00, 00, 00, time.UTC)
	tr.Started = time.Date(0001, 1, 1, 00, 00, 00, 00, time.UTC)
	tr.Status = common.TestRunStatusQueued
	tr.Details = ""
	tr.ExecutedCommands = []*common.ExecutedCommand{}

	if tr.ArchiverLogLevel == "" {
		tr.ArchiverLogLevel = "WARN"
	}
	if tr.ShardLogLevel == "" {
		tr.ShardLogLevel = "WARN"
	}
	if tr.SentinelLogLevel == "" {
		tr.SentinelLogLevel = "WARN"
	}
	if tr.AtomizerLogLevel == "" {
		tr.AtomizerLogLevel = "WARN"
	}
	if tr.WatchtowerLogLevel == "" {
		tr.WatchtowerLogLevel = "WARN"
	}
	if tr.LoadGenThreads == 0 {
		tr.LoadGenThreads = 1
	}

	tr.TerminateChan = make(chan bool, 1)
	tr.RetrySpawnChan = make(chan bool, 1)
	t.testRuns = append(t.testRuns, tr)
	t.ev <- coordinator.Event{
		Type: coordinator.EventTypeTestRunCreated,
		Payload: coordinator.TestRunCreatedPayload{
			Data: tr,
		},
	}
	t.PersistTestRun(tr)
}

// Scheduleris the main loop that checks if Queued testruns can commence
// execution by looking at the total number of active agents in the Running
// testruns, and considers vCPU limits on EC2 to prevent trying to start a test
// run for which the account does not have enough allowance
func (t *TestRunManager) Scheduler() {
	for {
		if t.loadComplete {
			// This is a janitor routine to look for running instances that are
			// not associated with a running testrun - they might have been left
			// running by mistake and should be terminated.
			instances := t.awsm.RunningInstances()
			killInstances := []*awsmgr.AwsInstance{}
			for _, i := range instances {
				testrunID := ""
				for _, t := range i.Instance.Tags {
					if *t.Key == "TestRunID" {
						testrunID = *t.Value
					}
				}

				if testrunID != "" {
					r, ok := t.GetTestRun(testrunID)
					if !ok {
						logging.Infof(
							"Instance %s (region %s) is active for an unknown testrun %s - killing",
							i.Instance.InstanceId,
							i.Region,
							testrunID,
						)
						killInstances = append(killInstances, i)
					} else {
						if r.Status != "Running" {
							logging.Infof("Instance %s (region %s) is active for testrun %s with status %s - killing", i.Instance.InstanceId, i.Region, testrunID, r.Status)
							killInstances = append(killInstances, i)
						}
					}
				}
			}
			if len(killInstances) > 0 {
				err := t.awsm.StopAgents(killInstances)
				if err != nil {
					logging.Warnf("Unable to stop agents: %v", err)
				}
			}
		}
		// Check if we are in maintenance mode - in that case we will not
		// start any test runs
		if !t.coord.GetMaintenance() {
			// Tally up all of the agents and vCPUs that we are currently using
			// with our active testruns (for which the agents have not been
			// stopped yet: a test run can be running (specifically: calculating
			// results) while the test agents have already been shut down)
			runningVCPUs := map[string]int32{}
			runningAgents := 0
			var nextQueued []*common.TestRun
			t.testRunsLock.Lock()
			for _, tr := range t.testRuns {
				if tr.Status == common.TestRunStatusRunning &&
					!tr.AWSInstancesStopped {
					runningVCPUsForTestRun := t.GetRequiredVCPUs(tr)
					for k, v := range runningVCPUsForTestRun {
						cur, ok := runningVCPUs[k]
						if !ok {
							runningVCPUs[k] = v
						} else {
							runningVCPUs[k] = cur + v
						}
					}
					runningAgents += len(tr.Roles)
				}
			}

			// Sort the queue by priority
			queue := t.testRuns[:]
			sort.Slice(queue, func(i, j int) bool {
				return queue[i].Priority > queue[j].Priority
			})

			for i, tr := range queue {
				if tr.Status == common.TestRunStatusQueued {

					// Test runs (specifically: time sweeps) can have a
					// configuration disallowing running the testrun before a
					// certain time - in which case this test run shouldn't be
					// considered for running
					canRun := tr.DontRunBefore.IsZero() ||
						tr.DontRunBefore.Before(time.Now())
					if !canRun {
						continue
					}

					// See how many VCPUs are needed for this testrun, and if
					// they fall within our allowed quota. If not, we cannot
					// consider this test run for execution
					requiredVCPUsForTestRun := t.GetRequiredVCPUs(tr)
					for k, v := range requiredVCPUsForTestRun {
						cur, ok := runningVCPUs[k]
						if !ok {
							if v > t.awsm.GetVCPULimit(k) {
								canRun = false
							}
						} else {
							if cur+v > t.awsm.GetVCPULimit(k) {
								canRun = false
							}
						}
					}
					if !canRun {
						logging.Infof(
							"Can't start test run %s because there's not enough capacity",
							tr.ID,
						)
						continue
					}

					// Check if executing this test would put the total number
					// of running agents over the configured limit. If this is
					// the case, we cannot consider this test for execution.
					if runningAgents+len(tr.Roles) > t.config.MaxAgents {
						logging.Infof(
							"Can't start test run %s because of the max agent limit",
							tr.ID,
						)
						continue
					}

					// It looks like we can start this test run within all
					// limiting parameters, so let's add it to the array of
					// test runs to execute, and update the tallied vCPU and
					// agent count for considering the next run to queue.
					for k, v := range requiredVCPUsForTestRun {
						cur, ok := runningVCPUs[k]
						if !ok {
							runningVCPUs[k] = v
						} else {
							runningVCPUs[k] = cur + v
						}
					}
					runningAgents += len(tr.Roles)
					nextQueued = append(nextQueued, t.testRuns[i])
				}
			}

			// If we have test runs to execute, execute them each in their own
			// little goroutine
			if len(nextQueued) > 0 {
				for i := range nextQueued {
					t.UpdateStatus(
						nextQueued[i],
						common.TestRunStatusRunning,
						"Executing test run ...",
					)
					go t.ExecuteTestRun(nextQueued[i])
				}
			}
			t.testRunsLock.Unlock()
		}
		time.Sleep(time.Second * 2)
	}
}

// Reschedule will insert a copy of the passed (failed) testrun into the queue
// if the max retries have not reached their limit. It will reset the agent IDs
// as well, since we need to spawn new AWS roles to run this test.
func (t *TestRunManager) Reschedule(tr *common.TestRun) {
	var newTr common.TestRun
	b, err := json.Marshal(tr)
	if err != nil {
		logging.Warnf("Could not marshal testruns: %v", err)
	}
	err = json.Unmarshal(b, &newTr)
	if err != nil {
		logging.Warnf("Could not unmarshal testruns: %v", err)
	}

	for i := range newTr.Roles {
		newTr.Roles[i].AgentID = -1
	}

	newTr.MaxRetries = newTr.MaxRetries - 1
	if newTr.MaxRetries > 0 {
		newTr.Priority = 3
		t.ScheduleTestRun(&newTr)
		t.UpdateStatus(
			&newTr,
			common.TestRunStatusQueued,
			"Requeued because of a failure",
		)
	}
}

// GetRequiredVCPUs will use the region and VCPU count of the chosen launch
// templates for all the roles in the test run to build a total tally map of
// region => vcpu_count and return it.
func (t *TestRunManager) GetRequiredVCPUs(tr *common.TestRun) map[string]int32 {
	ret := map[string]int32{}
	for i := range tr.Roles {
		lt, err := t.awsm.GetLaunchTemplate(tr.Roles[i].AwsLaunchTemplateID)
		if err == nil {
			key := fmt.Sprintf("%s-ondem", lt.Region)
			cur, ok := ret[key]
			if ok {
				ret[key] = cur + lt.VCPUCount
			} else {
				ret[key] = lt.VCPUCount
			}
		}
	}
	return ret
}
