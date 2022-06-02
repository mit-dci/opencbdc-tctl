package awsmgr

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/batch"
	"github.com/aws/aws-sdk-go-v2/service/batch/types"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

// seed_witcomm is the witness commitment that allows the seed_privkey to spend
// the output - should be the witness commitment matching the seed_privkey in
// ../testruns/parameters.go
var seed_witcomm = "6098c01c7a1a8f67a5e83ff49aa5488de5a3c867b81526e18040f5d6ec398446"
// TODO centralize this somewhere in stead of having two copies
var seed_privkey = "a0f36553548b3a66c003413140d7b59e43464ca11af66f25a6e746be501596b7"

type ShardSeed struct {
	CommitHash string `json:"commitHash"`
	SeedMode   int    `json:"mode"`
	Shards     int    `json:"shards"`
	Outputs    int    `json:"outputs"`
	TestRunID  string
	batchJobID string
}

func (am *AwsManager) GenerateSeed(seed ShardSeed) error {
	am.seedLock.Lock()
	defer am.seedLock.Unlock()
	alreadyHere, err := am.HasSeed(seed, true)
	if err != nil {
		return err
	}
	if !alreadyHere {
		input := &batch.SubmitJobInput{
			JobName:       aws.String(fmt.Sprintf("SEED_%s", seed.TestRunID)),
			JobQueue:      aws.String(os.Getenv("UHS_SEEDER_BATCH_JOB")),
			JobDefinition: aws.String(os.Getenv("UHS_SEEDER_BATCH_JOB")),
			ContainerOverrides: &types.ContainerOverrides{
				Environment: []types.KeyValuePair{
					{
						Name:  aws.String("SEED_SHARDS"),
						Value: aws.String(fmt.Sprintf("%d", seed.Shards)),
					},
					{
						Name:  aws.String("SEED_COMMIT"),
						Value: aws.String(seed.CommitHash),
					},
					{
						Name:  aws.String("SEED_OUTPUTS"),
						Value: aws.String(fmt.Sprintf("%d", seed.Outputs)),
					},
					{
						Name:  aws.String("SEED_VALUE"),
						Value: aws.String("1000000"), // Fixed
					},
					{
						Name:  aws.String("SEED_WITCOMM"),
						Value: aws.String(seed_witcomm),
					},
					{
						Name:  aws.String("SEED_PRIVATEKEY"),
						Value: aws.String(seed_privkey),
					},
					{
						Name:  aws.String("SEED_MODE"),
						Value: aws.String(fmt.Sprintf("%d", seed.SeedMode)),
					},
				},
			},
		}

		client, err := am.getBatchDefault()
		if err != nil {
			return err
		}

		out, err := client.SubmitJob(context.Background(), input)
		if err != nil {
			return err
		}

		seed.batchJobID = *out.JobId
		am.seeds = append(am.seeds, &seed)
	}
	return nil
}

// refreshSeedsLoop is launched in a separate goroutine when creating a new
// AwsManager. It will refresh the available shard seeds either when the user
// forces it through the forceRefreshSubnets channel, or after 5 minutes have
// passed and an active batch job is pending. No shard seeds appear out of
// nowhere, so if we're not generating it, the cache will not be stale.
func (am *AwsManager) refreshSeedsLoop() {
	for {
		refresh := false
		select {
		case <-am.forceRefreshSubnets:
			refresh = true
		case <-time.After(time.Minute * 5):
		}

		for _, s := range am.seeds {
			if s.batchJobID != "" {
				refresh = true
			}
		}
		if refresh {
			err := am.refreshSeeds()
			if err != nil {
				logging.Errorf("Error refreshing shard seeds: %v", err)
			}
		}
	}
}

// refreshSeeds will look at pending seed generations and check their status
// as well as load all seeds from S3
func (am *AwsManager) refreshSeeds() error {
	// Build a new array
	seeds := make([]*ShardSeed, 0)

	logging.Info("Refreshing seeds")

	// Check all entries with a batch job ID if the job completed
	input := &batch.DescribeJobsInput{
		Jobs: []string{},
	}
	for _, s := range am.seeds {
		if s.batchJobID != "" {
			input.Jobs = append(input.Jobs, s.batchJobID)
		}
	}
	jobCompleted := false
	if len(input.Jobs) > 0 {
		batchClient, err := am.getBatchDefault()
		if err != nil {
			return err
		}

		out, err := batchClient.DescribeJobs(context.Background(), input)
		if err != nil {
			return err
		}
		for _, job := range out.Jobs {
			if job.Status == types.JobStatusSucceeded {
				logging.Infof(
					"Batch job %s for shard seeding succeeded",
					*job.JobId,
				)
				jobCompleted = true
				continue
			}
			if job.Status == types.JobStatusFailed {
				logging.Warnf(
					"Batch job %s for shard seeding failed",
					*job.JobId,
				)
				continue
			}
			// Job is still busy, add to our new array
			for _, s := range am.seeds {
				if s.batchJobID == *job.JobId {
					seeds = append(seeds, s)
				}
			}
		}
	} else {
		logging.Info("No pending seed jobs")
	}
	if len(am.seeds) == 0 || jobCompleted {
		logging.Info("Loading seeds from S3")
		// Now, find all seeds in the S3 container
		region := os.Getenv("AWS_DEFAULT_REGION")
		if region == "" {
			logging.Warnf(
				"Environment AWS_DEFAULT_REGION is not set, falling back to us-east-1",
			)
			region = "us-east-1"
		}
		bucket := os.Getenv("BINARIES_S3_BUCKET")
		allSeeds, err := am.ListObjectsInS3(region, bucket, "shard-preseeds/")
		if err != nil {
			logging.Errorf("Error listing seeds in S3: %v", err)
			return err
		}

		for _, s := range allSeeds {
			s = s[15:] // lob off shard-preseeds/

			if !strings.Contains(s, ".tar") {
				continue
			}

			s = s[:len(s)-4] // lob off .tar

			mode := 0
			if strings.HasPrefix(s, "2pc_") {
				mode = 1
			}

			parts := strings.Split(s, "_")
			if len(parts) < 4 {
				continue
			}
			parts = parts[2:] // lob off "shard_preseed_"
			if mode == 1 {
				parts = parts[1:] // lob off 2pc_
			}

			if len(parts) < 4 { // outputs_start_end_commit
				continue
			}

			if parts[1] == "0" { // only consider the first in the series
				endFirstRange, err := strconv.Atoi(parts[2])
				if err != nil {
					continue
				}
				numShards := 256 / (endFirstRange + 1)
				numOutputs, err := strconv.Atoi(parts[0])
				if err != nil {
					continue
				}
				commitHash := parts[3]
				seeds = append(seeds, &ShardSeed{
					SeedMode:   mode,
					Outputs:    numOutputs,
					Shards:     numShards,
					CommitHash: commitHash,
				})
			}
		}
	}

	am.seeds = seeds
	return nil
}

func (s *ShardSeed) Equals(s2 *ShardSeed) bool {
	if s == nil && s2 == nil {
		return true
	}
	if (s == nil && s2 != nil) || (s != nil && s2 == nil) {
		return false
	}
	if s.Outputs == s2.Outputs && s.SeedMode == s2.SeedMode &&
		s.Shards == s2.Shards && s.CommitHash == s2.CommitHash {
		return true
	}
	return false
}

func (am *AwsManager) HasSeed(seed ShardSeed, pending bool) (bool, error) {
	refresh := false
	for _, s := range am.seeds {
		if s.batchJobID != "" {
			refresh = true
		}
	}
	if refresh {
		err := am.refreshSeeds()
		if err != nil {
			return false, err
		}
	}
	for _, s := range am.seeds {
		if s.Equals(&seed) && (s.batchJobID == "" || pending) {
			return true, nil
		}
	}
	return false, nil

}

func (am *AwsManager) GetAvailableSeeds() []*ShardSeed {
	seeds := make([]*ShardSeed, 0)
	for _, s := range am.seeds {
		if s.batchJobID == "" {
			seeds = append(seeds, s)
		}
	}

	return seeds
}
