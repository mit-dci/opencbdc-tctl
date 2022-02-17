package awsmgr

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

// TerminateSpotRequests will search for spot request(s) with the given tag and
// cancel them
func (awsm *AwsManager) CancelSpotRequests(
	clt *ec2.Client,
	spotRequestTag string,
) error {
	logging.Infof(
		"Checking for remaining spot requests with tag %s",
		spotRequestTag,
	)
	sirReq := &ec2.DescribeSpotInstanceRequestsInput{
		Filters: []types.Filter{
			{
				Name: aws.String(
					"tag:spot-request-tag",
				),
				Values: []string{
					spotRequestTag,
				},
			},
		},
	}
	var nextToken *string
	nextToken = nil
	cancelSpotRequests := []string{}

	for {
		sirReq.NextToken = nextToken
		sirResp, err := clt.DescribeSpotInstanceRequests(
			context.Background(),
			sirReq,
		)
		if err == nil {
			logging.Infof(
				"Found %d spot requests with tag %s",
				len(
					sirResp.SpotInstanceRequests,
				),
				spotRequestTag,
			)

			nextToken = sirResp.NextToken
			for _, sir := range sirResp.SpotInstanceRequests {
				if sir.State == types.SpotInstanceStateOpen {
					cancelSpotRequests = append(
						cancelSpotRequests,
						*sir.SpotInstanceRequestId,
					)
				}
			}
		} else {
			logging.Errorf("Failed to fetch open spot instance requests: %v", err)
		}
		if nextToken == nil {
			break
		}
	}

	if len(cancelSpotRequests) > 0 {
		logging.Infof(
			"Cancelling %d open spot instance requests",
			len(cancelSpotRequests),
		)
		cancelReq := &ec2.CancelSpotInstanceRequestsInput{
			SpotInstanceRequestIds: cancelSpotRequests,
		}
		cancelResp, err := clt.CancelSpotInstanceRequests(
			context.Background(),
			cancelReq,
		)
		if err == nil {
			logging.Infof(
				"Succesfully canceled %d open spot instance requests",
				len(
					cancelResp.CancelledSpotInstanceRequests,
				),
			)
		} else {
			logging.Errorf("Failed to cancel open spot instance requests: %v", err)
		}

	} else {
		logging.Infof("No open spot instance requests to cancel")
	}
	return nil
}
