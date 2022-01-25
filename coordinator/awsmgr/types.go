package awsmgr

import "github.com/aws/aws-sdk-go-v2/service/ec2/types"

// AwsLaunchTemplate describes an AWS EC2 launch template
type AwsLaunchTemplate struct {
	Region       string `json:"region"`
	TemplateID   string `json:"id"`
	Description  string `json:"description"`
	InstanceType string `json:"instanceType"`
	VCPU         string `json:"vCPU"`
	VCPUCount    int32
	RAM          string `json:"ram"`
	Bandwidth    string `json:"bandwidth"`
}

// AwsSubnet describes an AWS EC2 subnet
type AwsSubnet struct {
	Region   string `json:"region"`
	SubnetID string `json:"id"`
	AZ       string `json:"az"`
	Name     string `json:"name"`
}

// AwsInstance describes a running AWS EC2 instance
type AwsInstance struct {
	Region   string
	Instance *types.Instance
}

type StartAgent struct {
	TemplateID  string
	Count       int
	InstanceIDs []string
}
