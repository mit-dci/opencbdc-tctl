package awsmgr

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

// AwsManager is the main type for managing AWS resources from the coordinator
type AwsManager struct {
	Enabled               bool
	runningInstances      []*AwsInstance
	runningInstancesLock  sync.Mutex
	forceRefreshTemplates chan bool
	launchTemplates       []AwsLaunchTemplate
	forceRefreshSubnets   chan bool
	subnets               []AwsSubnet
	vcpuLimit             map[string]int32
	s3Clients             map[string]*s3.Client
	s3ClientsLock         sync.Mutex
	seeds                 []*ShardSeed
	forceRefreshSeeds     chan bool
	seedLock              sync.Mutex
}

// NewAwsManager creates a new AwsManager instance
func NewAwsManager() *AwsManager {
	am := &AwsManager{
		s3ClientsLock:         sync.Mutex{},
		s3Clients:             map[string]*s3.Client{},
		vcpuLimit:             map[string]int32{},
		Enabled:               true,
		runningInstancesLock:  sync.Mutex{},
		seedLock:              sync.Mutex{},
		runningInstances:      make([]*AwsInstance, 0),
		forceRefreshTemplates: make(chan bool, 1),
		forceRefreshSubnets:   make(chan bool, 1),
		seeds:                 make([]*ShardSeed, 0),
		forceRefreshSeeds:     make(chan bool, 1),
	}
	// Run initialization in a separate goroutine
	go func() {
		// Load the default AWS/EC2 configuration - this to check that we
		// can properly instantiate EC2 clients and that we have the necessary
		// credentials to do so. If this fails, we cannot use AWS at all and
		// should set enabled to false for the AWS Manager
		_, err := config.LoadDefaultConfig(
			context.Background(),
			config.WithDefaultRegion("us-east-1"),
			defaultRetrier(),
		)
		if err != nil {
			logging.Warnf("Could not initialize AWS: %v", err)
			am.Enabled = false
			return
		}

		// Refresh the running instances from EC2
		i, err := am.refreshRunningInstances()
		if err != nil {
			logging.Warnf("Could not initialize AWS: %v", err)
			am.Enabled = false
			return
		}
		am.runningInstancesLock.Lock()
		am.runningInstances = i
		am.runningInstancesLock.Unlock()

		// If we have instances running upon startup, these are left over
		// from when the controller was previously running. Most likely due to
		// it crashing in the middle of executing tests. Clean up these
		// instances now since we cannot use them and we don't want them to
		// incurr further running costs
		if len(am.runningInstances) > 0 {
			logging.Warnf(
				"There were instances running when the controller started up - probably left overs from an active run while the controller rebooted or crashed. Killing all instances.",
			)
			err = am.KillAllInstances()
			if err != nil {
				logging.Warnf("Could not kill instances: %v", err)
			}
		}

		// Load the launch templates, subnets, seeds and limits for the first
		// time
		am.refreshLaunchTemplates()
		am.refreshSubnets()
		am.refreshLimits()
		am.refreshSeeds()
	}()

	// Start a loop that refreshes the launch templates periodically
	go am.refreshLaunchTemplatesLoop()

	// Start a loop that refreshes the subnets periodically
	go am.refreshSubnetsLoop()

	// Start a loop that refreshes the quotas/limits periodically
	go am.refreshLimitsLoop()

	return am
}
