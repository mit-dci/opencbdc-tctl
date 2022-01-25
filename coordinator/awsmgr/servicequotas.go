package awsmgr

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

// getSQ creates a servicequotas client for the given region and returns it. If
// an error occurs with the configuration of the client, it will return that
// error and a nil client
func (am *AwsManager) getSQ(region string) (*servicequotas.Client, error) {
	if !am.Enabled {
		return nil, errors.New("AWS not enabled")
	}
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		defaultRetrier(),
	}
	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	return servicequotas.NewFromConfig(cfg), nil
}

// refreshLimitsLoop will be executed in a separate goroutine and will take care
// of refetching the servicequotas from AWS every 15 minutes
func (am *AwsManager) refreshLimitsLoop() {
	for {
		time.Sleep(time.Minute * 15)
		am.refreshLimits()
	}
}

// refreshLimits will use the servicequota api to determine what our vCPU limits
// are in each region. These limits are used to determine if we're able to
// execute a certain test that's queued. If the required VCPUs exceed our
// alotted service quota, we will wait for enough capacity to be available (by
// other test runs completing first, for instance, or the service quota being
// increased) before commencing a test
func (am *AwsManager) refreshLimits() {
	newLimits := map[string]int32{}
	mtx := sync.Mutex{}
	err := am.RunSQForAllRegions(
		func(sq *servicequotas.Client, r string) error {
			var nextToken *string
			nextToken = nil
			for {
				input := &servicequotas.ListServiceQuotasInput{
					ServiceCode: aws.String("ec2"),
					NextToken:   nextToken,
				}
				output, err := sq.ListServiceQuotas(context.Background(), input)

				if err != nil {
					logging.Errorf("Error fetching service quota: %v", err)
					return err
				} else {
					nextToken = output.NextToken
					for _, q := range output.Quotas {
						if *q.QuotaCode == "L-34B43A08" {
							mtx.Lock()
							newLimits[fmt.Sprintf("%s-spot", r)] = int32(*q.Value)
							mtx.Unlock()
						}
						if *q.QuotaCode == "L-1216C47A" {
							mtx.Lock()
							newLimits[fmt.Sprintf("%s-ondem", r)] = int32(*q.Value)
							mtx.Unlock()
						}
					}
				}

				if nextToken == nil {
					break
				}
			}
			return nil
		},
	)
	if err == nil {
		am.vcpuLimit = newLimits
	}
	logging.Infof("vCPU Limits [%d]:", len(newLimits))
	for k, v := range newLimits {
		logging.Infof("%s %d", k, v)
	}
}

// GetVCPULimit will return the limit of number of VCPUs for a certain `key`
// where key is `<region>-[spot|ondem]`, for instance 'us-east-1-spot' describes
// how many vcpus we can launch in spot markets in region us-east-1
func (am *AwsManager) GetVCPULimit(key string) int32 {
	return am.vcpuLimit[key]
}

// RunSQForAllRegions will execute a function for each enabled region, passing
// in an SQ client configured for that region and wait for the function to
// complete for all regions. If an error occurs for any region, it is returned
// to the caller
func (am *AwsManager) RunSQForAllRegions(
	f func(e *servicequotas.Client, r string) error,
) error {
	regions := am.GetAllRegions()
	wg := sync.WaitGroup{}
	errs := make([]error, 0)
	for _, rg := range regions {
		wg.Add(1)
		go func(reg string) {
			defer wg.Done()
			e, err := am.getSQ(reg)
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
