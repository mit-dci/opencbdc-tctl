package awsmgr

import (
	"context"
	"errors"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

// getBatchDefault returns a Batch client for use in the default region.
func (am *AwsManager) getBatchDefault() (*batch.Client, error) {
	batchRegion := os.Getenv("AWS_DEFAULT_REGION")
	if batchRegion == "" {
		logging.Warnf(
			"Environment AWS_DEFAULT_REGION is not set, falling back to us-east-1",
		)
		batchRegion = "us-east-1"
	}
	return am.getBatch(batchRegion)
}

// getBatch returns a Batch client for use in region `region`. This default
// factory
// has no debugging enabled
func (am *AwsManager) getBatch(region string) (*batch.Client, error) {
	return am.getBatchWithDebug(region, false)
}

// getBatchWithDebug creates a Batch client for use in region `region` with
// optionally enabled debugging which would log the request and response bodies
// of REST calls to the console. This is very extensive, so should only be used
// in extreme debugging scenarios. If you want to enable debugging, set `debug`
// to true.
func (am *AwsManager) getBatchWithDebug(
	region string,
	debug bool,
) (*batch.Client, error) {
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
	return batch.NewFromConfig(cfg), nil
}
