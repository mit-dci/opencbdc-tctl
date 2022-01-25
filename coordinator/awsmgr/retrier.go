package awsmgr

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
)

// retryableErrorCodes constains a customized list of error codes we are willing
// to retry the request for
var retryableErrorCodes = map[string]struct{}{
	"RequestTimeout":          {},
	"RequestTimeoutException": {},

	// Throttled status codes
	"Throttling":                             {},
	"ThrottlingException":                    {},
	"ThrottledException":                     {},
	"RequestThrottledException":              {},
	"TooManyRequestsException":               {},
	"ProvisionedThroughputExceededException": {},
	"TransactionInProgressException":         {},
	"RequestLimitExceeded":                   {},
	"BandwidthLimitExceeded":                 {},
	"LimitExceededException":                 {},
	"RequestThrottled":                       {},
	"SlowDown":                               {},
	"PriorRequestNotComplete":                {},
	"EC2ThrottledException":                  {},
}

// retriables contains a custom list of conditions we can retry the request for
var retriables = []retry.IsErrorRetryable{
	retry.NoRetryCanceledError{},
	retry.RetryableError{},
	retry.RetryableConnectionError{},
	retry.RetryableErrorCode{
		Codes: retryableErrorCodes,
	},
}

// defaultRetrier is the retrier we inject to the EC2 clients we create, which
// allows us more fine-grained control over which errors we want to have the AWS
// client retry natively, and which ones it should just bubble up to the caller.
// In our spawning logic, we don't want the AWS client to keep retrying certain
// errors (such as capacity problems) - we prefer to cycle to the next
// availability zone or switch from Spot to OnDemand launches to ensure a timely
// launch of all our instances.
func defaultRetrier() config.LoadOptionsFunc {
	return config.WithRetryer(func() aws.Retryer {
		return retry.NewStandard(func(o *retry.StandardOptions) {
			o.MaxAttempts = 10
			o.MaxBackoff = time.Second * 3
			o.Retryables = retriables
		})
	})
}
