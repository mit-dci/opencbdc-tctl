package testruns

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/coordinator/awsmgr"
	"github.com/mit-dci/opencbdc-tctl/logging"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// PreseedShards will instruct the agents that run shard roles to download the
// relevant shard preseed set from S3 and unpack it into the right spot for the
// shard to pick it up on startup.
func (t *TestRunManager) PreseedShards(
	tr *common.TestRun,
	envs map[int32][]byte,
) error {
	if !tr.PreseedShards {
		return nil
	}

	// Get the right preseed file based on the architecture and type of shard.
	// In-memory shards (default for 2PC, tested for Atomizer) have a flat file
	// with a set of 32-byte values that indicate the UHSs in the system.
	// On-disk shards (atomizer) require preseeds with a compressed LevelDB that
	// has the predefined UHS set incorporated.
	filePrefix := "shard_preseed_"
	targetPath := "db.tar"
	tarCreateNoDir := false
	shards := t.GetAllRolesSorted(tr, common.SystemRoleShard)
	if t.Is2PC(tr.Architecture) {
		targetPath = "shard_preseed_%SHARDIDX%_%SHARDNODEIDX%.tar"
		filePrefix = "2pc_shard_preseed_"
		tarCreateNoDir = true
		shards = t.GetAllRolesSorted(tr, common.SystemRoleShardTwoPhase)
	}
	t.UpdateStatus(tr, common.TestRunStatusRunning, "Pre-seeding shards")
	t.WriteLog(tr, "Pre-seeding shards")
	wg := sync.WaitGroup{}
	errs := make([]error, 0)
	errsLock := sync.Mutex{}

	sourceRegion := os.Getenv("AWS_DEFAULT_REGION")
	if sourceRegion == "" {
		logging.Warnf(
			"Environment AWS_DEFAULT_REGION is not set, falling back to us-east-1",
		)
		sourceRegion = "us-east-1"
	}

	bucket := os.Getenv("BINARIES_S3_BUCKET")
	preseedShard := func(r *common.TestRunRole, trn *common.TestRun, shardStart, shardEnd int, utxoCount int64, envID []byte) {
		agentID := r.AgentID

		// Get the correct source file based on the utxo count and the shard
		// range and seeder commithash
		sourcePath := fmt.Sprintf(
			"shard-preseeds/%s%d_%d_%d_%s.tar",
			filePrefix,
			utxoCount,
			shardStart,
			shardEnd,
			tr.SeederHash,
		)
		// The target path can be determined (for the 2pc shards this is the
		// case) by the cluster and node index of the shard role
		substitutedTargetPath := t.SubstituteParameters(
			[]string{targetPath},
			r,
			trn,
		)

		t.WriteLog(
			tr,
			"Pre-seeding agent %d path %s with [%s]/%s/%s",
			agentID,
			substitutedTargetPath[0],
			sourceRegion,
			bucket,
			sourcePath,
		)

		// Instruct the agent to deploy the file from S3 and unpack it
		res, err := t.am.QueryAgentWithTimeout(
			agentID,
			&wire.DeployFileFromS3RequestMsg{
				EnvironmentID: envID,
				SourceBucket:  bucket,
				SourceRegion:  sourceRegion,
				SourcePath:    sourcePath,
				TargetPath:    substitutedTargetPath[0],
				Unpack:        true,
				FlatUnpack:    true,
				UnpackNoDir:   tarCreateNoDir,
			},
			10*time.Minute,
		)

		if err != nil {
			errsLock.Lock()
			err = fmt.Errorf(
				"error seeding agent %d (AWS Instance %s): %v",
				agentID,
				r.AwsAgentInstanceId,
				err,
			)
			errs = append(errs, err)
			errsLock.Unlock()
		} else {
			_, ok := res.(*wire.DeployFileFromS3ResponseMsg)
			if !ok {
				errs = append(errs, fmt.Errorf("error seeding agent %d (AWS Instance %s): Unexpected return type. Expected DeployFileFromS3ResponseMsg, got %T", agentID, r.AwsAgentInstanceId, res))
			}
		}

		// 2PC shard uses a single file, not a folder - have to rename
		// the file coming from the tar archive
		if t.Is2PC(trn.Architecture) {
			inmemPreseedSource := fmt.Sprintf(
				"2pc_shard_preseed_%d_%d_%d",
				utxoCount,
				shardStart,
				shardEnd,
			)

			inmemPreseedTarget := t.SubstituteParameters(
				[]string{"shard_preseed_%SHARDIDX%_%SHARDNODEIDX%"},
				r,
				trn,
			)

			t.WriteLog(
				tr,
				"Renaming in-memory shard preseed file from %s to %s on agent %d",
				inmemPreseedSource,
				inmemPreseedTarget[0],
				agentID,
			)
			res, err := t.am.QueryAgentWithTimeout(
				agentID,
				&wire.RenameFileRequestMsg{
					EnvironmentID: envID,
					SourcePath:    inmemPreseedSource,
					TargetPath:    inmemPreseedTarget[0],
				},
				15*time.Second,
			)
			
			if err != nil {
				t.WriteLog(tr, "WARN: Renaming preseed failed, could be because of newer preseeding already having the correct filename. May ignore: %v", err)
			} else {
				_, ok := res.(*wire.RenameFileResponseMsg)
				if !ok {
					errs = append(errs, fmt.Errorf("error seeding agent %d (AWS Instance %s): Unexpected return type. Expected DeployFileFromS3ResponseMsg, got %T", agentID, r.AwsAgentInstanceId, res))
				}
			}
		}

		t.WriteLog(tr, "Pre-seeding agent %d succeeded", agentID)

		wg.Done()
	}

	shardClusters := len(shards) / tr.ShardReplicationFactor
	shardRange := 256 / shardClusters
	// Call preseedShard for each of the shard roles with the correct range
	// which is dictated by the architecture
	if t.IsAtomizer(tr.Architecture) {
		for _, r := range shards {
			start := 0 + (r.Index%shardClusters)*shardRange
			end := (((r.Index % shardClusters) + 1) * shardRange) - 1
			if (r.Index % shardClusters) == shardClusters-1 {
				end = 255
			}
			wg.Add(1)
			go preseedShard(r, tr, start, end, tr.PreseedCount, envs[r.AgentID])
		}
	} else if t.Is2PC(tr.Architecture) {
		for _, r := range shards {
			cluster := r.Index / tr.ShardReplicationFactor
			start := 0 + (cluster%shardClusters)*shardRange
			end := (((cluster % shardClusters) + 1) * shardRange) - 1
			if (cluster % shardClusters) == shardClusters-1 {
				end = 255
			}
			wg.Add(1)
			go preseedShard(r, tr, start, end, tr.PreseedCount, envs[r.AgentID])
		}
	}
	wg.Wait()
	if len(errs) > 0 {
		jointErr := ""
		for _, e := range errs {
			jointErr += e.Error() + "\n"
		}
		return errors.New("Failed to seed shards: " + jointErr)
	}
	t.UpdateStatus(tr, common.TestRunStatusRunning, "Done pre-seeding shards")

	return nil
}

func (t *TestRunManager) CheckPreseed(tr *common.TestRun) error {
	if !tr.PreseedShards {
		return nil
	}

	seedMode := 0
	numShards := len(
		t.GetAllRolesSorted(tr, common.SystemRoleShard),
	) / tr.ShardReplicationFactor
	if t.Is2PC(tr.Architecture) {
		seedMode = 1
		numShards = len(
			t.GetAllRolesSorted(tr, common.SystemRoleShardTwoPhase),
		) / tr.ShardReplicationFactor
	}
	var err error

	wantSeed := awsmgr.ShardSeed{
		Outputs:    int(tr.PreseedCount),
		SeedMode:   seedMode,
		Shards:     numShards,
		CommitHash: tr.SeederHash,
		TestRunID:  tr.ID,
	}
	hasSeed, err := t.awsm.HasSeed(wantSeed, false)
	if err != nil {
		return fmt.Errorf("error checking preseed existence: %v", err)
	}

	if !hasSeed {
		t.UpdateStatus(tr, common.TestRunStatusRunning, "Generating preseed")
		hasSeed, err := t.awsm.HasSeed(wantSeed, true)
		if err != nil {
			return fmt.Errorf("error checking preseed existence: %v", err)
		}
		if !hasSeed {
			err := t.awsm.GenerateSeed(wantSeed)
			if err != nil {
				return fmt.Errorf("error generating preseed: %v", err)
			}
		}
	}

	start := time.Now()
	for {
		if time.Since(start).Minutes() > 15 {
			return fmt.Errorf(
				"Shard preseeding timed out - "+
					"Please check on Batch job with name [SEED_%s] from the AWS console for details",
				tr.ID,
			)
		}
		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			"Waiting for preseed generation to complete",
		)
		hasSeed, err := t.awsm.HasSeed(wantSeed, false)
		if err != nil {
			return fmt.Errorf("error checking preseed existence: %v", err)
		}
		if hasSeed {
			break
		}

		time.Sleep(time.Second * 5)
	}

	return nil
}
