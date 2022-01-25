package awsmgr

import (
	"context"
	"errors"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// getEC2 returns an EC2 client for use in region `region`. This default factory
// has no debugging enabled
func (am *AwsManager) getEC2(region string) (*ec2.Client, error) {
	return am.getEC2WithDebug(region, false)
}

// getEC2WithDebug creates an EC2 client for use in region `region` with
// optionally enabled debugging which would log the request and response bodies
// of REST calls to the console. This is very extensive, so should only be used
// in extreme debugging scenarios. If you want to enable debugging, set `debug`
// to true.
func (am *AwsManager) getEC2WithDebug(
	region string,
	debug bool,
) (*ec2.Client, error) {
	if !am.Enabled {
		return nil, errors.New("AWS not enabled")
	}
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		defaultRetrier(),
	}
	if debug {
		opts = append(
			opts,
			config.WithClientLogMode(
				aws.LogRequestWithBody|aws.LogResponseWithBody,
			),
		)
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return ec2.NewFromConfig(cfg), nil
}

// RunEC2ForAllRegions will execute a function for each enabled region, passing
// in an EC2 client configured for that region and wait for the function to
// complete for all regions. If an error occurs for any region, it is returned
// to the caller
func (am *AwsManager) RunEC2ForAllRegions(
	f func(e *ec2.Client, r string) error,
) error {
	regions := am.GetAllRegions()
	wg := sync.WaitGroup{}
	errs := make([]error, 0)
	for _, rg := range regions {
		wg.Add(1)
		go func(reg string) {
			defer wg.Done()
			e, err := am.getEC2(reg)
			if err != nil {
				errs = append(errs, err)
				return
			}
			err = f(e, reg)
			if err != nil {
				errs = append(errs, err)
				return
			}
		}(rg)
	}
	wg.Wait()
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
