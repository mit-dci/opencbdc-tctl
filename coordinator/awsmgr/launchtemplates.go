package awsmgr

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

// ForceRefreshLaunchTemplates is a method to force refreshing the launch
// templates. This is currently unused but could be hooked up to a REST API to
// allow refreshing this by force from the UI
func (am *AwsManager) ForceRefreshLaunchTemplates() {
	am.forceRefreshTemplates <- true
}

// refreshLaunchTemplates will query the available launch templates in AWS EC2
// and store them in the AwsManager. They can then be used from methods like
// `GetLaunchTemplate()` and `LaunchTemplates()`
func (am *AwsManager) refreshLaunchTemplates() {
	// Create a new array to not overwrite the one still being read by clients
	newLaunchTemplates := make([]AwsLaunchTemplate, 0)
	// A mutex to guard the newLaunchTemplates array, since the
	// RunEC2ForAllRegions will execute for each region in parallel
	mtx := sync.Mutex{}
	// Read launch template from all regions
	err := am.RunEC2ForAllRegions(func(e *ec2.Client, r string) error {
		// Use nextToken to fetch more pages in a paginated result in case we
		// end up having tons of launch templates
		var nextToken *string
		nextToken = nil
		for {
			input := &ec2.DescribeLaunchTemplatesInput{
				NextToken: nextToken,
			}

			output, err := e.DescribeLaunchTemplates(
				context.Background(),
				input,
			)

			if err != nil {
				return err
			} else {
				nextToken = output.NextToken
				for _, l := range output.LaunchTemplates {
					// Translate the resulting AWS objects into our own native
					// AwsLaunchTemplate object
					lt := AwsLaunchTemplate{TemplateID: *l.LaunchTemplateId, Region: r}
					for _, t := range l.Tags {
						if *t.Key == "Interface_Description" {
							lt.Description = *t.Value
						}
					}

					// Parse the specs from the text of the launch template
					// description
					startSpecs := strings.Index(lt.Description, "(")
					if startSpecs > -1 {
						specsString := lt.Description[startSpecs+1 : len(lt.Description)-2]
						specs := strings.Split(specsString, "/")
						lt.InstanceType = strings.TrimSpace(lt.Description[:startSpecs])
						lt.VCPU = strings.TrimSpace(specs[0])

						vcpuCnt, _ := strconv.ParseInt(strings.Replace(strings.ToLower(lt.VCPU), "vcpu", "", 1), 10, 32)
						lt.VCPUCount = int32(vcpuCnt)
						lt.RAM = strings.TrimSpace(specs[1])
						lt.Bandwidth = strings.TrimSpace(specs[2])
					}

					// Insert the template into our array
					mtx.Lock()
					newLaunchTemplates = append(newLaunchTemplates, lt)
					mtx.Unlock()
				}
			}

			if nextToken == nil {
				break
			}
		}
		return nil
	})
	// If no errors occurred, adopt the new array
	if err == nil {
		am.launchTemplates = newLaunchTemplates
	}
}

// refreshLaunchTemplatesLoop will be ran in a goroutine after creation of a new
// AwsManager. It will refresh the templates either on a forcefully called
// refresh through the forceRefreshTemplates channel, or after 15 minutes.
func (am *AwsManager) refreshLaunchTemplatesLoop() {
	for {
		select {
		case <-am.forceRefreshTemplates:
		case <-time.After(time.Minute * 15):
		}

		am.refreshLaunchTemplates()

	}
}

// GetLaunchTemplateRegion will return the AWS region for a given launch
// template
func (am *AwsManager) GetLaunchTemplateRegion(launchTemplateID string) string {
	for _, lt := range am.launchTemplates {
		if lt.TemplateID == launchTemplateID {
			return lt.Region
		}
	}
	return "unknown"
}

// LaunchTemplates will return the available launch templates or an empty array
// if AWS was not enabled
func (am *AwsManager) LaunchTemplates() []AwsLaunchTemplate {
	if !am.Enabled {
		return []AwsLaunchTemplate{}
	} else {
		return am.launchTemplates
	}
}

// GetLaunchTemplate returns a single launch template identified by its ID or an
// error if it was not found in the set of available launch templates
func (am *AwsManager) GetLaunchTemplate(
	templateID string,
) (AwsLaunchTemplate, error) {
	for _, t := range am.launchTemplates {
		if t.TemplateID == templateID {
			return t, nil
		}
	}
	return AwsLaunchTemplate{}, fmt.Errorf("template %s not found", templateID)
}
