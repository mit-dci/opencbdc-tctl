package testruns

import (
	"errors"
	"fmt"
	"time"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/coordinator/awsmgr"
)

// KillAwsAgents will terminate all running EC2 instances for the specified
// test run
func (t *TestRunManager) KillAwsAgents(tr *common.TestRun) error {
	// SkipCleanup is a setting that can be configured from the UI, which should
	// prevent terminating the instances. This should be used cautiously, but
	// can be helpful to determine causes of problems in the tests, by being
	// allowed to connect to the test agent's instance after the test and
	// inspect it.
	if tr.SkipCleanUp {
		t.WriteLog(
			tr,
			"SkipCleanup enabled - not killing AWS agents, clean up manually!",
		)
		return nil
	}

	t.WriteLog(tr, "Killing all AWS agents")
	// Read all instance IDs from the testrun data (the AWS instance IDs are
	// saved in the role data) - and then call a method on the AWS Manager to
	// stop the actual instances
	ids := make([]string, 0)
	for _, r := range tr.Roles {
		if r.AwsAgentInstanceId != "" {
			ids = append(ids, r.AwsAgentInstanceId)
		}
	}
	t.WriteLog(tr, "Killing %d agents", len(ids))
	return t.awsm.StopAgentsByInstanceIds(ids)
}

// HasAWSRoles will return true if the test run has roles that (are supposed to)
// run on AWS EC2
func (t *TestRunManager) HasAWSRoles(tr *common.TestRun) bool {
	for _, r := range tr.Roles {
		if r.AwsLaunchTemplateID != "" {
			return true
		}
	}

	return false
}

// RetrySpawn is used to manually initiate respawning of the AWS roles that are
// not online yet
func (t *TestRunManager) RetrySpawn(id string) {
	for _, tr := range t.testRuns {
		if tr.ID == id {
			select {
			case tr.RetrySpawnChan <- true:
			case <-time.After(time.Second * 1):
				//timeout
			}
		}
	}
}

// SpawnAWSInstances will look for the roles needed to be spawned on AWS EC2 and
// initiate spawning them
func (t *TestRunManager) SpawnAWSInstances(tr *common.TestRun) bool {
	t.WriteLog(tr, "(Re)spawning AWS Instances")
	killInstances := []string{}

	spawnIndexes := []int{}
	spawnInstances := []string{}

	// First, idenfity all test run roles that have no agent ID assigned
	// (meaning they are not connected to the controller yet), but do have an
	// AWS instance ID assigned (which is the instance that we're waiting to
	// connect to the controller to play that role in our test). These are
	// instances that we are waiting for - and we kill them to spawn new
	// instances for these roles.
	for i, r := range tr.Roles {
		if r.AgentID == -1 {
			// This agent is not connected to the controller yet
			if tr.Roles[i].AwsAgentInstanceId != "" {
				// We are already waiting for this agent to connect from a
				// specific AWS instance ID. Kill it to retry spawning the role.
				// There is a setting "KeepTimedOutAgents" available in the UI
				// to not kill those failed roles, which can aid in debugging
				// reasons why spawned roles weren't able to connect to the
				// controller - keeping them online allows us to connect to it
				// and poke around to see what happened.
				if !tr.KeepTimedOutAgents {
					killInstances = append(
						killInstances,
						tr.Roles[i].AwsAgentInstanceId,
					)
				}
			}

			// This is an agent that we still need (either first or retrying
			// attempt). Append to the array(s) of instances to spawn
			spawnIndexes = append(spawnIndexes, i)
			spawnInstances = append(
				spawnInstances,
				tr.Roles[i].AwsLaunchTemplateID,
			)
		}
	}

	// If we need to kill instances we are retrying, do so now.
	if len(killInstances) > 0 {
		err := t.awsm.StopAgentsByInstanceIds(killInstances)
		if err != nil {
			t.WriteLog(
				tr,
				"Error stopping %d instances we are retrying to spawn: %v",
				len(killInstances),
				err,
			)
		}
	}

	// Call StartNewAgents on the AWS manager to do the actual API calls to AWS
	// EC2 and boot up the compute instances. We get a list of instances back in
	// the same order as we requested them
	instances, errs := t.awsm.StartNewAgents(spawnInstances, tr.ID)
	if len(errs) > 0 {
		// If something went wrong during spawning the roles, abort the test run
		jointErr := ""
		for _, e := range errs {
			jointErr += e.Error() + "\n"
		}
		t.FailTestRun(tr, errors.New("Failed to spawn AWS roles: "+jointErr))

		// Find any non-nil instance in the return array, which are roles that
		// were already spawned when the error occurred. We need to stop them
		// because we won't be using them for the test.
		nonNilInstances := []*awsmgr.AwsInstance{}
		for _, i := range instances {
			if i != nil {
				nonNilInstances = append(nonNilInstances, i)
			}
		}
		if len(nonNilInstances) > 0 {
			t.WriteLog(
				tr,
				"Stopping %d already spawned instances",
				len(nonNilInstances),
			)
			err := t.awsm.StopAgents(nonNilInstances)
			if err != nil {
				t.WriteLog(tr, "Error stopping instances: %v", err)
			}
		}
		return false
	}

	// spawnIndexes contains the indexes within the role set of the instances we
	// spawned. Since the return array from StartNewAgents is guaranteed to be
	// in the same order, we can assign the spawned indexes to the role array
	// easily. We use the AwsAgentInstanceId to monitor the progress of the
	// spawned agents connecting to the controller
	for i, idx := range spawnIndexes {
		tr.Roles[idx].AwsAgentInstanceId = *instances[i].Instance.InstanceId
	}

	return true
}

// WaitForAWSInstances will use the spawned instance IDs set to the role
// information by SpawnAWSInstances to determine if all the roles needed for
// the test run are online and ready to begin the test. It will retry spawning
// if it takes too long, and fail if it doesn't succeed after three retries.
func (t *TestRunManager) WaitForAWSInstances(
	tr *common.TestRun,
) (bool, bool, bool) {

	// Keep track of time and number of retries
	respawnTried := 0
	start := time.Now()
	for {
		// t.ShouldTerminate will return true if the user has manually
		// terminated the test run while we were waiting for the roles to come
		// online
		if t.ShouldTerminate(tr) {
			return false, false, true
		}

		// Get all the agent data from the coordinator - this is information
		// from all connected agents. The agent will send the coordinator its
		// AWS instance ID as part of the system information that gets sent on
		// first connection. The startup script for the agent determines the EC2
		// Instance ID and passes it as environment variable to the agent
		// binary, that will then send it to the controller. We can use this to
		// match the connecting agent against the role that's waiting for the
		// given instance ID because we have assigned the instance ID to the
		// role metadata. If we find a match, we assign the agent ID to the
		// role, such that we know which agent to instruct to run that
		// particular role.
		for _, a := range t.coord.GetAgents() {
			if a.SystemInfo.AWS {
				for i, r := range tr.Roles {
					if r.AgentID == -1 &&
						r.AwsAgentInstanceId == a.SystemInfo.EC2InstanceID {
						tr.Roles[i].AgentID = a.ID
					}
				}
			}
		}

		// Count the number of roles we're still waiting for. If we have been
		// waiting for longer than two minutes, print out the instance IDs we're
		// waiting for to the test run log. This can help the user to find the
		// instance in the AWS console and figure out why it's not online yet.
		waiting := 0
		for _, r := range tr.Roles {
			if r.AgentID == -1 {
				if time.Since(start).Minutes() > 2 {
					t.WriteLog(
						tr,
						"Role %s %d still waits for AWS agent %s",
						r.Role,
						r.Index,
						r.AwsAgentInstanceId,
					)
				}
				waiting++
			}
		}

		// If there's no roles we're waiting for anymore, that means
		// everything's ready to go - so return from the routine here
		if waiting == 0 {
			return false, false, false
		}

		// Update the status to let the frontend user know how much roles are
		// still pending
		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			fmt.Sprintf("Waiting for %d AWS agents to come online", waiting),
		)

		if time.Since(start).Minutes() > 5 {
			// It sometimes happens that these roles don't fully start up -
			// meaning the instance is booted succesfully (that EC2 API call
			// succeeds), but the agent does not end up connecting to the
			// controller. In stead of failing the test run at this point,
			// we will retry spawning the roles that are not online yet.
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				fmt.Sprintf(
					"%d roles are still not online after 5 minutes. Respawning them",
					waiting,
				),
			)

			if respawnTried < 3 {
				// Retry spawning. Reset the start time and call
				// SpawnAWSInstances again
				start = time.Now()
				respawnTried++
				if !t.SpawnAWSInstances(tr) {
					return true, false, false
				}
			} else {
				// We have retried spawning these roles three times and they're
				// still not online. Give up at this point. Important to note is
				// that all roles that are online are now waiting, doing
				// nothing, but are accumulating AWS costs. Especially in
				// situations where a large number of roles are online and only
				// 1-2 are pending, this is very costly and so we shouldn't keep
				// waiting or retrying indefinitely.
				return false, true, false
			}

		}

		// At the end of the check loop, see if the user has manually initiated
		// retrying the spawning of roles.
		select {
		case <-tr.RetrySpawnChan:
			// Retry spawning on user request. Reset the start time and call
			// SpawnAWSInstances again
			start = time.Now()
			if !t.SpawnAWSInstances(tr) {
				return true, false, false
			}
		case <-time.After(time.Second * 5):
		}
	}
}
