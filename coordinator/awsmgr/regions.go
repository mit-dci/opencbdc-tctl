package awsmgr

// The regions we use in AWS. For now this is hard coded but we could add some
// form of discovery later on
var regions = []string{"us-east-1", "us-east-2", "us-west-2"}

// GetAllRegions returns a list of available regions. This now returns the hard
// coded array, but in the future the array could be periodically re-read using
// whatever discovery method we find for the regions we want enabled.
func (am *AwsManager) GetAllRegions() []string {
	return regions
}
