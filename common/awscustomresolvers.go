package common

import (
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3customResolver returns a function which can provide an S3 VPC interface
// endpoint value for the AWS SDK config provided the S3_INTERFACE_ENDPOINT
// and S3_INTERFACE_REGION environment variables are set. If they are not set,
// the configuration should default to the public S3 endpoint.
func S3customResolver() aws.EndpointResolver {
	cr := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			s3InterfaceEndpoint := os.Getenv("S3_INTERFACE_ENDPOINT")
			s3InterfaceRegion := os.Getenv("S3_INTERFACE_REGION")
			if s3InterfaceEndpoint != "" && s3InterfaceRegion != "" {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           s3InterfaceEndpoint,
					SigningRegion: s3InterfaceRegion,
				}, nil
			} else {
				// Fallback to default resolution
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			}
		}
		// Fallback to default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	return cr
}
