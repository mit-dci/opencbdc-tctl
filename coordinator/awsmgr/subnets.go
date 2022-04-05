package awsmgr

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// ForceRefreshSubnets is a method to force refreshing the subnets
// This is currently unused but could be hooked up to a REST API to
// allow refreshing this by force from the UI
func (am *AwsManager) ForceRefreshSubnets() {
	am.forceRefreshSubnets <- true
}

// refreshSubnets will load all subnets from EC2. This is used to determine the
// availability zones we can use for launching instances, given that each AZ
// have their own subnet
func (am *AwsManager) refreshSubnets() {
	newSubnets := make([]AwsSubnet, 0)
	mtx := sync.Mutex{}
	err := am.RunEC2ForAllRegions(func(e *ec2.Client, r string) error {
		var nextToken *string
		nextToken = nil
		for {
			input := &ec2.DescribeSubnetsInput{
				NextToken: nextToken,
			}

			output, err := e.DescribeSubnets(context.Background(), input)

			if err != nil {
				return err
			} else {
				nextToken = output.NextToken
				for _, s := range output.Subnets {
					sn := AwsSubnet{SubnetID: *s.SubnetId, Region: r, AZ: *s.AvailabilityZone}
					for _, t := range s.Tags {
						if *t.Key == "Name" {
							sn.Name = *t.Value
						}
					}
					// TODO: make this configurable?
					if strings.Contains(sn.Name, "hamilton-private-") {
						mtx.Lock()
						newSubnets = append(newSubnets, sn)
						mtx.Unlock()
					}
				}
			}

			if nextToken == nil {
				break
			}
		}
		return nil
	})
	if err == nil {
		am.subnets = newSubnets
	}
}

// refreshSubnetsLoop is launched in a separate goroutine when creating a new
// AwsManager. It will refresh the available subnets either when the user forces
// it through the forceRefreshSubnets channel, or after 15 minutes have passed
func (am *AwsManager) refreshSubnetsLoop() {
	for {
		select {
		case <-am.forceRefreshSubnets:
		case <-time.After(time.Minute * 15):
		}

		am.refreshSubnets()
	}
}

// GetSubnetsForRegion returns the subnets available in the given region
func (am *AwsManager) GetSubnetsForRegion(region string) []AwsSubnet {
	ret := make([]AwsSubnet, 0)
	for _, sn := range am.subnets {
		if sn.Region == region {
			ret = append(ret, sn)
		}
	}
	return ret
}
