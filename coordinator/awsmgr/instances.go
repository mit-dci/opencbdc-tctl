package awsmgr

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

// RunningInstances returns the currently running instances in EC2
func (am *AwsManager) RunningInstances() []*AwsInstance {
	if !am.Enabled {
		return []*AwsInstance{}
	}
	return am.runningInstances
}

// refreshRunningInstances refreshes the running instances from EC2. This is
// normally only done on start-up, since the controller is the only entity
// launching new instances. As such, the instances array will be maintained by
// the logic that starts and stops new instances in stead of polling the data
// from EC2
func (am *AwsManager) refreshRunningInstances() ([]*AwsInstance, error) {
	if !am.Enabled {
		return []*AwsInstance{}, nil
	}

	// Load all instances from EC2
	allInstances := make([]*AwsInstance, 0)
	instancesLock := sync.Mutex{}
	err := am.RunEC2ForAllRegions(func(e *ec2.Client, region string) error {
		var nextToken *string
		for {
			res, err := e.DescribeInstances(
				context.Background(),
				&ec2.DescribeInstancesInput{NextToken: nextToken},
			)
			if err != nil {
				return err
			}
			instancesLock.Lock()
			for i := range res.Reservations {
				for j := range res.Reservations[i].Instances {
					allInstances = append(
						allInstances,
						&AwsInstance{
							Region:   region,
							Instance: &res.Reservations[i].Instances[j],
						},
					)
				}
			}
			instancesLock.Unlock()
			if res.NextToken != nil {
				nextToken = res.NextToken
			} else {
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	logging.Infof(
		"[AWS Manager] Found %d instances, filtering by state 'Running' and name tag 'test-agent-*' or 'test-controller-agent'",
		len(allInstances),
	)

	// Filtering the instances by the proper name prefix - to prevent us from
	// meddling with instances that are spawned manually or for instance the
	// bastion host
	result := make([]*AwsInstance, 0)
	for _, i := range allInstances {
		if i.Instance.State.Name == types.InstanceStateNameRunning {
			for _, t := range i.Instance.Tags {
				if *t.Key == "Name" {
					if strings.HasPrefix(*t.Value, "test-agent-") {
						result = append(result, i)
					}
					if *t.Value == "test-controller-agent" { // Default name when tagging failed
						result = append(result, i)
					}
				}
			}
		}
	}

	logging.Infof(
		"[AWS Manager] Found %d instances running and with name tag 'test-agent-*' or 'test-controller-agent'",
		len(result),
	)

	return result, nil

}

// KillAllInstances will terminate all known running instances in EC2
func (am *AwsManager) KillAllInstances() error {
	if !am.Enabled {
		return nil
	}

	if len(am.runningInstances) == 0 {
		return nil
	}

	return am.StopAgents(am.runningInstances)

}

// StopAgentsByInstanceIds will terminate the EC2 instances specified in the ids
// array. It will enumerate the runningInstances and match their ID to the IDs
// in the passed array, and then call StopAgents()
func (am *AwsManager) StopAgentsByInstanceIds(ids []string) error {
	agents := make([]*AwsInstance, 0)
	for _, i := range am.runningInstances {
		for _, id := range ids {
			if *i.Instance.InstanceId == id {
				agents = append(agents, i)
			}
		}
	}

	return am.StopAgents(agents)
}

// StopAgents will terminate the EC2 instances by the instance objects passed
func (am *AwsManager) StopAgents(a []*AwsInstance) error {
	logging.Infof("Stopping %d instances...", len(a))
	// Run this logic separately for all regions
	err := am.RunEC2ForAllRegions(func(e *ec2.Client, region string) error {
		// Build an array of instances in this region to terminate
		killInstances := make([]*AwsInstance, 0)
		for _, i := range a {
			if i.Region == region {
				killInstances = append(killInstances, i)
			}
		}

		// Terminate the instances in batches of 50 not to exceed API limits
		for i := 0; i < len(killInstances); i += 50 {
			killIds := []string{}
			end := i + 50
			if end > len(killInstances) {
				end = len(killInstances)
			}
			for _, j := range killInstances[i:end] {
				killIds = append(killIds, *j.Instance.InstanceId)
			}

			logging.Infof(
				"Stopping %d instances in region %s",
				len(killIds),
				region,
			)

			input := &ec2.TerminateInstancesInput{
				InstanceIds: killIds,
			}

			_, err := e.TerminateInstances(context.Background(), input)
			if err != nil {
				log.Printf("Error terminating instances: %v", err)
				return err
			}

			for _, j := range killInstances[i:end] {
				j.Instance.State = &types.InstanceState{
					Name: types.InstanceStateNameTerminated,
				}
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// If everything went well, we can update the runningInstances array by
	// removing the instances that we just terminated. Effectively we're
	// building a new array in which we insert all instances we did not touch
	am.runningInstancesLock.Lock()
	newArr := make([]*AwsInstance, 0)
	for _, i := range am.runningInstances {
		if i.Instance.State == nil ||
			i.Instance.State.Name != types.InstanceStateNameTerminated {
			newArr = append(newArr, i)
		}
	}
	logging.Infof(
		"Running instances array went from %d to %d",
		len(am.runningInstances),
		len(newArr),
	)
	am.runningInstances = newArr
	am.runningInstancesLock.Unlock()
	return nil
}
