package awsmgr

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

// StartNewAgents is the main logic that spawns our new instances. We pass it an
// array of templateIDs to launch. For each instance we want to launch there is
// one entry in the array. So there can be many entries of the same launch
// template in this array - each entry in the list will result in an instance
// launched. The return array will contain the instances that we launched in the
// exact same sequence as the templateIDs that were passed.
func (am *AwsManager) StartNewAgents(
	templateIDs []string,
	testRunID string,
) ([]*AwsInstance, []error) {

	returnVal := make([]*AwsInstance, len(templateIDs))

	// Sort templates to launch by region and template ID. This allows us to
	// launch the templates in batches per region and per template ID
	launches := map[string][]*StartAgent{}
	for _, tid := range templateIDs {
		launchTemplate, err := am.GetLaunchTemplate(tid)
		if err != nil {
			return nil, []error{err}
		}

		arr, ok := launches[launchTemplate.Region]
		if ok {
			found := false
			for i, sa := range arr {
				if sa.TemplateID == tid {
					arr[i].Count++
					found = true
					break
				}
			}
			if !found {
				arr = append(arr, &StartAgent{TemplateID: tid, Count: 1})
			}
		} else {
			arr = []*StartAgent{{TemplateID: tid, Count: 1}}
		}
		launches[launchTemplate.Region] = arr
	}

	errs := make([]error, 0)
	errsLock := sync.Mutex{}
	wg := sync.WaitGroup{}
	for k, v := range launches {
		wg.Add(1)
		go func(region string, agents []*StartAgent) {
			// This subroutine will need to launch all templates defined in the
			// agents array - this contains for each template (that's guaranteed
			// to be defined within this region) how many instances of that
			// template need to be launched

			// First, we create an EC2 client in the correct region
			clt, err := am.getEC2(region)
			if err != nil {
				errsLock.Lock()
				errs = append(errs, err)
				errsLock.Unlock()
				wg.Done()
				return
			}

			// The marketOptionsList is a convenience array that effectively has
			// two entries: Spot market and On Demand. By turning this into an
			// array we can easily loop over the spot-ondemand options in a
			// for{} loop
			marketOptionsList := []*types.InstanceMarketOptionsRequest{
				{
					MarketType: types.MarketTypeSpot,
					SpotOptions: &types.SpotMarketOptions{
						BlockDurationMinutes:         aws.Int32(60),
						InstanceInterruptionBehavior: types.InstanceInterruptionBehaviorTerminate,
						SpotInstanceType:             types.SpotInstanceTypeOneTime,
					},
				},
				nil,
			}

			// Get all subnets in this region, which we'll need to determine the
			// availablity zones in which we can launch instances
			subnets := am.GetSubnetsForRegion(region)
			for _, ag := range agents {
				// ag now specifies a specific instance template (ag.TemplateID)
				// we have to launch in this region, and how many instances we
				// need (ag.Count)

				done := false

				// First, we generate a random agent name for each agent we're
				// launching
				agentNames := make([]string, ag.Count)
				for i := range agentNames {
					randName := make([]byte, 8)
					_, err = rand.Read(randName[:])
					if err != nil {
						errsLock.Lock()
						errs = append(errs, err)
						errsLock.Unlock()
						wg.Done()
						return
					}
					agentNames[i] = fmt.Sprintf("test-agent-%x", randName)
				}

				// We loop over all market options and availability zones six
				// times. The reason for retrying is that instance availability
				// is very fluctuating - we can fail to launch in us-east-1a the
				// first time, but after trying all other AZs it could suddenly
				// succeed because other AWS clients have terminated their
				// instances.
				for attempt := 0; attempt < 6; attempt++ {
					if done {
						break
					}
					if attempt > 0 {
						// If this is not our first attempt, wait a bit before
						// retrying to increase our odds that the capacity has
						// changed
						logging.Infof(
							"[Region %s] Still need %d agents, waiting for 15 seconds to retry",
							region,
						)
						time.Sleep(
							time.Second * 15,
						)
					}
					for _, marketOptions := range marketOptionsList {
						// marketOptions now contains either the Spot market
						// (first cycle) or On-Demand (second cycle)
						if done {
							break
						}
						for _, sn := range subnets {
							// sn now contains the subnet, through which we can
							// fetch the availability zone to pass to
							// RunInstances
							if done {
								break
							}
							for {
								if done {
									break
								}

								// Launch max 50 at a time to prevent hitting
								// limits
								max := int32(ag.Count)
								if max > int32(50) {
									max = 50
								}

								logging.Infof(
									"[Region %s] Launching maximum %d instances of template [%s] with subnet [%s], AZ [%s] and spot [%t]",
									region,
									ag.Count,
									ag.TemplateID,
									sn.SubnetID,
									sn.AZ,
									marketOptions != nil,
								)

								// Build the RunInstancesInput that we'll fire
								// at the EC2 API. This defines the subnet, the
								// market options (spot/ondemand), the template
								// ID and the amount of instances we need.
								input := &ec2.RunInstancesInput{
									LaunchTemplate: &types.LaunchTemplateSpecification{
										LaunchTemplateId: &ag.TemplateID,
									},
									InstanceMarketOptions: marketOptions,
									NetworkInterfaces: []types.InstanceNetworkInterfaceSpecification{
										{
											SubnetId:    &sn.SubnetID,
											DeviceIndex: aws.Int32(0),
										},
									},
									Placement: &types.Placement{
										AvailabilityZone: &sn.AZ,
									},
									MinCount: aws.Int32(1),
									MaxCount: aws.Int32(max),
								}
								// Create a random ID for the spot request
								spotRequestTag, err := common.RandomID(12)
								if err != nil {
									errsLock.Lock()
									errs = append(
										errs,
										fmt.Errorf(
											"error fetching Random ID: %v",
											err,
										),
									)
									errsLock.Unlock()
									wg.Done()
									return
								}
								if marketOptions != nil {
									// If creating a spot request, tag it - this
									// way we can query the spot request later
									// and ensure it does not remain dorment
									input.TagSpecifications = []types.TagSpecification{
										{
											ResourceType: types.ResourceTypeSpotInstancesRequest,
											Tags: []types.Tag{
												{
													Key: aws.String(
														"spot-request-tag",
													),
													Value: &spotRequestTag,
												},
											},
										},
									}
								}

								launched := int32(0)
								result, err := clt.RunInstances(
									context.Background(),
									input,
								)
								if err != nil {
									// Log the error, but continue the loop - we
									// might succceed in other regions/spot
									// settings
									logging.Warnf(
										"There was a RunInstances failure in region %s for testrun %s: %v",
										region,
										testRunID,
										err,
									)

									// Also check if any spot requests were left
									// open, and kill them
									go func() {
										// Wait for a second just to ensure any
										// spot instance requests that are
										// somehow still
										// in transit are committed, so we don't
										// miss them.
										time.Sleep(time.Second * 1)
										err := am.CancelSpotRequests(
											clt,
											spotRequestTag,
										)
										if err != nil {
											logging.Errorf(
												"Error canceling spot requests: %v",
												err,
											)
										}
									}()
								} else {
									// The lauches were succesful!
									launched = int32(len(result.Instances))
									logging.Infof("[Region %s] Launched %d instances of template [%s] with subnet [%s] and spot [%b]", region, len(result.Instances), ag.TemplateID, sn.SubnetID, marketOptions != nil)
									if len(result.Instances) > 0 {
										am.runningInstancesLock.Lock()
										for i := range result.Instances {
											// Assign tags here so that they're
											// part of our cache
											key := "Name"
											key2 := "TestRunID"
											result.Instances[i].Tags = []types.Tag{
												{
													Key:   &key,
													Value: &agentNames[i],
												},
												{
													Key:   &key2,
													Value: &testRunID,
												},
											}
											// Assign this instance to the first
											// occurrence of a matching
											// template ID for which no instance
											// has yet been assigned in the
											// return array.
											for j := range templateIDs {
												if templateIDs[j] == ag.TemplateID && returnVal[j] == nil {
													logging.Debugf("Assigning instance index %d, id %s to result %d", i, *result.Instances[i].InstanceId, j)
													// Since there is only one
													// goroutine per region, and
													// instance templates exist
													// in only one region, this
													// array can be safely
													// modified from each
													// region-goroutine without
													// using a mutex
													returnVal[j] = &AwsInstance{Region: region, Instance: &result.Instances[i]}
													// Append the instance to
													// the global running
													// instances array
													am.runningInstances = append(am.runningInstances, returnVal[j])
													break
												}
											}
										}
										am.runningInstancesLock.Unlock()

										// Deduct the number of instances we
										// launched from the number we still
										// need
										ag.Count -= len(result.Instances)
										if ag.Count == 0 {
											done = true
										}
									}
								}
								if launched < max {
									// If we got less instances than we
									// requested as maximum, it's time to try
									// another AZ or market
									break
								}
							}
						}
					}
				}

				// At this point we either completed launching all instances, or
				// we ran through the six retry cycles. If there are still
				// instances we need, then the capacity is not available on EC2
				// or our quota were exceeded
				if ag.Count > 0 {
					errsLock.Lock()
					errs = append(
						errs,
						errors.New("was unable to launch enough instances"),
					)
					errsLock.Unlock()
					wg.Done()
					return
				}

				// Tag the instances in a separate goroutine - we already stored
				// the tags in the instance data we keep in memory - this
				// routine just applies them to the actual AWS infrastructure
				// using the CreateTags API
				go func() {
					// Make an array with all the tags we need to still apply
					retryTagging := make([]*types.Instance, 0)
					for _, r := range returnVal {
						if r != nil && r.Region == region {
							retryTagging = append(retryTagging, r.Instance)
						}
					}

					previousLen := 0
					failed := 0
					for len(retryTagging) > 0 {
						// Keep looping until we have finished all tags
						logging.Debugf(
							"[Region %s] Tagging %s instances",
							region,
							len(retryTagging),
						)
						// If the number of instances left to tag is the same
						// as the previous loop, then there is a continous
						// failure. If this happens three times abandon the loop
						// and stop attempting to tag the resources. This is
						// unlikely to happen, but we want to make sure not to
						// be stuck in this loop forever
						if len(retryTagging) == previousLen {
							failed++
							if failed > 3 {
								logging.Errorf(
									"Continuous failures tagging resources. %d left untagged.",
									len(retryTagging),
								)
								return
							}
						}
						previousLen = len(retryTagging)
						// Loop over the array back to front such that we don't
						// invalidate the index by removing the succesfully
						// tagged instances from the array
						for i := len(retryTagging) - 1; i >= 0; i-- {
							// Make a call to the CreateTags API for each
							// individual instance we want to tag. We can't
							// combine this in a single API call since that can
							// only be done for applying the same tag to a set
							// of resources. Unfortunately, since each test
							// agent has a different Name tag, this cannot be
							// used.
							r := retryTagging[i]
							tagInput := &ec2.CreateTagsInput{
								Resources: []string{*r.InstanceId},
								Tags:      r.Tags, // Use our local saved cache
							}
							_, err := clt.CreateTags(
								context.Background(),
								tagInput,
							)
							if err == nil {
								// Tagging succeeded, remove this element from
								// the array
								retryTagging[len(retryTagging)-1], retryTagging[i] = retryTagging[i], retryTagging[len(retryTagging)-1]
								retryTagging = retryTagging[:len(retryTagging)-1]
							} else {
								// Tagging failed, keep the element in the array
								// such that we retry it in the next loop cycle
								logging.Errorf("Failed tagging resource: %v", err)
							}
							// Sleep to prevent excessive API calling and
							// hitting limits
							time.Sleep(time.Millisecond * 200)
						}
						time.Sleep(time.Second * 2)
					}
				}()
			}
			wg.Done()
		}(k, v)
	}
	// WaitGroup contains an entry for each region in which we launch resources,
	// wait for all goroutines (per region) to complete and then return
	wg.Wait()

	return returnVal, errs
}
