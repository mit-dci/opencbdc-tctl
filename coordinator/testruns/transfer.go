package testruns

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/coordinator/sources"
	"github.com/mit-dci/opencbdc-tctl/logging"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// copyFiles describes which files to copy from the given system role upon
// completion of the test
var copyFiles = map[common.SystemRole][]string{
	common.SystemRoleArchiver: {
		"tp_samples.txt",
		"block_log.txt%%OPT",
	},
	common.SystemRoleRaftAtomizer: {
		"tp_samples.txt%%OPT",
		"block_log.txt%%OPT",
		"discarded_log.txt%%OPT",
		"complete_tx_log.txt%%OPT",
		"state_machine_log.txt%%OPT",
		"raft_store_log.txt%%OPT",
		"tx_notify_log.txt%%OPT",
	},
	common.SystemRoleAtomizerCliWatchtower: {
		"latency_samples_%IDX%.txt%%OPT",
		"tx_samples_%IDX%.txt%%OPT",
		"telemetry.bin%%OPT",
	},
	common.SystemRoleSentinel: {},
	common.SystemRoleShard: {
		"tp_samples.txt%%OPT",
		"block_log.txt%%OPT",
	},
	common.SystemRoleCoordinator: {
		"telemetry.bin%%OPT",
	},
	common.SystemRoleWatchtower: {
		"tp_samples.txt%%OPT",
		"block_log.txt%%OPT",
	},
	common.SystemRoleShardTwoPhase: {
		"telemetry.bin%%OPT",
	},
	common.SystemRoleSentinelTwoPhase: {
		"telemetry.bin%%OPT",
	},
	common.SystemRoleTwoPhaseGen: {
		"tx_samples_%IDX%.txt",
		"tps_target_%IDX%.txt%%OPT",
		"telemetry.bin%%OPT",
	},
	common.SystemRoleParsecGen: {
		"tx_samples_%IDX%.txt",
		"telemetry.bin%%OPT",
	},
	common.SystemRoleAgent: {
		"telemetry.bin%%OPT",
	},
	common.SystemRoleRuntimeLockingShard: {
		"telemetry.bin%%OPT",
	},
}

// CopyOutputs will use the `copyFiles` map to instruct the agents to upload all
// indicated files from its file system to S3 so that the coordinator can
// download them later
func (t *TestRunManager) CopyOutputs(
	tr *common.TestRun,
	envs map[int32][]byte,
	ignoreErrors bool,
) error {
	path := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/outputs", tr.ID),
	)

	t.UpdateStatus(
		tr,
		common.TestRunStatusRunning,
		"Uploading testrun output files from agents to S3 (0%)",
	)

	allDownloads := make([]common.S3Download, 0)
	allDownloadsLock := sync.Mutex{}

	f := func(role *common.TestRunRole) error {
		if len(copyFiles[common.SystemRole(role.Role)]) > 0 {
			for _, f := range t.SubstituteParameters(copyFiles[common.SystemRole(role.Role)], role, tr) {
				// Optionally ignore failures for files that may or may not
				// exist on the target agent
				ignoreFile := false
				if strings.HasSuffix(f, "%%OPT") {
					f = f[:len(f)-5]
					ignoreFile = true
				}

				// Calculate the target path in the outputs bucket based on the
				// test run ID, system role, index and filename
				targetPath := fmt.Sprintf(
					"testruns/%s/outputs/%s-%d-%s",
					tr.ID,
					string(role.Role),
					role.Index,
					f,
				)

				// Instruct the agent to upload the file to S3
				msg, err := t.am.QueryAgentWithTimeout(
					role.AgentID,
					&wire.UploadFileToS3RequestMsg{
						EnvironmentID: envs[role.AgentID],
						SourcePath:    f,
						TargetRegion:  os.Getenv("AWS_REGION"),
						TargetBucket:  os.Getenv("OUTPUTS_S3_BUCKET"),
						TargetPath:    targetPath,
					},
					3*time.Minute,
				)
				err = t.processS3UploadResponse(role.AgentID, msg, err)
				if err != nil {
					// Watchtower CLI temporarily ignored due to new tx_samples
					// usage (optional)
					if ignoreErrors || ignoreFile {
						logging.Warnf(
							"Ignoring error while copying file %s from agent %d: %v",
							f,
							role.AgentID,
							err,
						)
						continue
					}
					return err
				}
				// Append this uploaded file to the array of downloads
				allDownloadsLock.Lock()
				allDownloads = append(allDownloads, common.S3Download{
					TargetPath: filepath.Join(
						path,
						fmt.Sprintf(
							"%s-%d-%s",
							string(role.Role),
							role.Index,
							f,
						),
					),
					SourceRegion: os.Getenv("AWS_REGION"),
					SourceBucket: os.Getenv("OUTPUTS_S3_BUCKET"),
					SourcePath:   targetPath,
					Retries:      10,
				})
				allDownloadsLock.Unlock()

			}
		}
		return nil
	}
	err := t.RunForAllAgents(
		f,
		tr,
		"Uploading testrun output files from agents to S3",
		time.Minute*10,
	)
	if err != nil {
		return err
	}

	// Append all downloads to the downloads array for the test run. This is
	// done to defer the downloading until after the roles are shut down, not
	// keeping them online longer than necessary. Once everything is in S3 we
	// can download that after the roles have been shut down.
	tr.PendingResultDownloads = append(
		tr.PendingResultDownloads,
		allDownloads...)
	return nil
}

// GetPerformanceProfiles instructs the agents to upload the performance data
// gathered while running the command(s) to S3
func (t *TestRunManager) GetPerformanceProfiles(
	tr *common.TestRun,
	cmds []runningCommand,
	envs map[int32][]byte,
) error {
	errs := make([]error, 0)

	// The path where we store all performance data in S3 derive from the test
	// run ID
	path := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/performanceprofiles", tr.ID),
	)
	err := os.MkdirAll(path, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	t.WriteLog(
		tr,
		"Instructing agents to upload their performance profiles to S3",
	)
	total := len(cmds)
	done := int32(0)
	wg := sync.WaitGroup{}
	wg.Add(total)
	allDownloads := make([]common.S3Download, 0)
	allDownloadsLock := sync.Mutex{}
	for _, c := range cmds {
		go func(cmd runningCommand, complete *int32) {
			// The regular performance counters that are always gathered
			files := []string{fmt.Sprintf("perf_%x.txt", cmd.commandID)}
			if tr.RunPerf {
				// These files are only created when we run the command in
				// `perf`
				files = append(
					files,
					fmt.Sprintf("perf_%x.data", cmd.commandID),
				)
				files = append(
					files,
					fmt.Sprintf("perf_%x.script", cmd.commandID),
				)
				files = append(
					files,
					fmt.Sprintf("perf_%x.data.tar.bz2", cmd.commandID),
				)
			}
			for _, f := range files {
				// Instruct the agent to upload the file
				msg, err := t.am.QueryAgentWithTimeout(
					cmd.agentID,
					&wire.UploadFileToS3RequestMsg{
						EnvironmentID: envs[cmd.agentID],
						SourcePath:    f,
						TargetRegion:  os.Getenv("AWS_REGION"),
						TargetBucket:  os.Getenv("OUTPUTS_S3_BUCKET"),
						TargetPath: fmt.Sprintf(
							"testruns/%s/performanceprofiles/%s",
							tr.ID,
							f,
						),
					},
					30*time.Second,
				)
				// If anything went wrong, append it to the errors array and
				// stop trying further uploads
				err = t.processS3UploadResponse(cmd.agentID, msg, err)
				if err != nil {
					errs = append(errs, err)
					break
				}

				// Append the uploaded files to the list of things to download
				// once we're done
				allDownloadsLock.Lock()
				allDownloads = append(allDownloads, common.S3Download{
					TargetPath:   filepath.Join(path, f),
					SourceRegion: os.Getenv("AWS_REGION"),
					SourceBucket: os.Getenv("OUTPUTS_S3_BUCKET"),
					SourcePath: fmt.Sprintf(
						"testruns/%s/performanceprofiles/%s",
						tr.ID,
						f,
					),
					Retries: 10,
				})
				allDownloadsLock.Unlock()

			}
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				fmt.Sprintf(
					"Copying performance profiles (%.1f%%)",
					float64(atomic.AddInt32(complete, 1))/float64(total)*50,
				),
			)
			wg.Done()
		}(c, &done)
	}
	wg.Wait()
	if len(errs) > 0 {
		for _, e := range errs {
			logging.Errorf("%v", e)
		}
		logging.Warnf(
			"%d errors occurred while copying performance data",
			len(errs),
		)
		t.WriteLog(
			tr,
			"%d errors occurred while copying performance data",
			len(errs),
		)
	}

	// Append all downloads to the downloads array for the test run.
	tr.PendingResultDownloads = append(
		tr.PendingResultDownloads,
		allDownloads...)
	return nil
}

// GetPerformanceProfiles instructs the agents to upload the performance data
// gathered while running the command(s) to S3
func (t *TestRunManager) GetLogFiles(
	tr *common.TestRun,
	cmds []runningCommand,
	envs map[int32][]byte,
) error {
	// TODO: refactor with GetPerformanceProfiles and CopyOutputs
	errs := make([]error, 0)

	// The path where we store all performance data in S3 derive from the test
	// run ID
	path := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s/logs", tr.ID),
	)
	err := os.MkdirAll(path, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}

	t.WriteLog(
		tr,
		"Instructing agents to upload their log files to S3",
	)
	total := len(cmds)
	done := int32(0)
	wg := sync.WaitGroup{}
	wg.Add(total)
	allDownloads := make([]common.S3Download, 0)
	allDownloadsLock := sync.Mutex{}
	for _, c := range cmds {
		go func(cmd runningCommand, complete *int32) {
			// The regular performance counters that are always gathered
			files := []string{
				fmt.Sprintf("command_%x_stdout.txt", cmd.commandID),
				fmt.Sprintf("command_%x_stderr.txt", cmd.commandID),
			}
			for _, f := range files {
				// Instruct the agent to upload the file
				msg, err := t.am.QueryAgentWithTimeout(
					cmd.agentID,
					&wire.UploadFileToS3RequestMsg{
						EnvironmentID: envs[cmd.agentID],
						SourcePath:    f,
						TargetRegion:  os.Getenv("AWS_REGION"),
						TargetBucket:  os.Getenv("OUTPUTS_S3_BUCKET"),
						TargetPath: fmt.Sprintf(
							"testruns/%s/logs/%s",
							tr.ID,
							f,
						),
					},
					120*time.Second,
				)
				// If anything went wrong, append it to the errors array and
				// stop trying further uploads
				err = t.processS3UploadResponse(cmd.agentID, msg, err)
				if err != nil {
					errs = append(errs, err)
					break
				}

				// Append the uploaded files to the list of things to download
				// once we're done
				allDownloadsLock.Lock()
				allDownloads = append(allDownloads, common.S3Download{
					TargetPath:   filepath.Join(path, f),
					SourceRegion: os.Getenv("AWS_REGION"),
					SourceBucket: os.Getenv("OUTPUTS_S3_BUCKET"),
					SourcePath: fmt.Sprintf(
						"testruns/%s/logs/%s",
						tr.ID,
						f,
					),
					Retries: 10,
				})
				allDownloadsLock.Unlock()

			}
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				fmt.Sprintf(
					"Copying log files (%.1f%%)",
					float64(atomic.AddInt32(complete, 1))/float64(total)*50,
				),
			)
			wg.Done()
		}(c, &done)
	}
	wg.Wait()
	if len(errs) > 0 {
		for _, e := range errs {
			logging.Errorf("%v", e)
		}
		logging.Warnf(
			"%d errors occurred while copying log files",
			len(errs),
		)
		t.WriteLog(
			tr,
			"%d errors occurred while copying log files",
			len(errs),
		)
	}

	// Append all downloads to the downloads array for the test run.
	tr.PendingResultDownloads = append(
		tr.PendingResultDownloads,
		allDownloads...)
	return nil
}

// RedownloadTestOutputsFromS3 will enumerate all files in the S3 bucket for
// the given testrun and download them all. This can be triggered from the user
// interface. This exists because sometimes files get either corrupted or fail
// downloading and redoing the downloads can help fix that.
func (t *TestRunManager) RedownloadTestOutputsFromS3(tr *common.TestRun) error {
	downloads := make([]common.S3Download, 0)
	prefix := fmt.Sprintf("testruns/%s/", tr.ID)

	for _, bucket := range []string{os.Getenv("OUTPUTS_S3_BUCKET"), os.Getenv("BINARIES_S3_BUCKET")} {
		objects, err := t.awsm.ListObjectsInS3(
			os.Getenv("AWS_REGION"),
			bucket,
			prefix,
		)
		if err != nil {
			return err
		}

		for _, o := range objects {
			downloads = append(downloads, common.S3Download{
				Retries:      10,
				SourceRegion: os.Getenv("AWS_REGION"),
				SourceBucket: bucket,
				SourcePath:   o,
				TargetPath:   filepath.Join(common.DataDir(), o),
			})
		}
	}
	logging.Infof(
		"Re-downloading %d outputs from S3 for testrun %s",
		len(downloads),
		tr.ID,
	)

	return t.awsm.DownloadMultipleFromS3(downloads)
}

// BinariesExistInS3 checks existence and returns an empty string
// if not, and the path in S3 if it does.
func (t *TestRunManager) BinariesExistInS3(
	tr *common.TestRun,
	seeder bool,
) (string, error) {
	hash := tr.CommitHash
	debug := tr.RunPerf || tr.Debug
	if seeder {
		hash = tr.SeederHash
		debug = false
	}
	binariesInS3 := fmt.Sprintf("binaries/%s.tar.gz", hash)
	if debug {
		binariesInS3 = fmt.Sprintf("binaries/%s-debug.tar.gz", hash)
	}
	exist, err := t.awsm.FileExistsOnS3(os.Getenv("AWS_REGION"),
		os.Getenv("BINARIES_S3_BUCKET"),
		binariesInS3)
	if !exist || err != nil {
		return "", err
	}
	return binariesInS3, nil
}

// UploadBinaries upload binaries for this testrun to S3
func (t *TestRunManager) UploadBinaries(
	tr *common.TestRun,
	seeder bool,
) (string, error) {

	hash := tr.CommitHash
	debug := tr.RunPerf || tr.Debug
	if seeder {
		hash = tr.SeederHash
		debug = false
	}
	sourcePath, err := sources.BinariesArchivePath(
		hash,
		debug,
	)
	if err != nil {
		return "", err
	}

	binariesInS3 := fmt.Sprintf("binaries/%s.tar.gz", hash)
	if debug {
		// We need a separate archive for debug binaries since they perform
		// much worse. We can't run debugging or perf on a binary set with
		// optimizations because the stacktraces won't make much sense.
		binariesInS3 = fmt.Sprintf("binaries/%s-debug.tar.gz", hash)
	}
	_, loaded := t.pendingBinaryUploads.LoadOrStore(binariesInS3, true)
	if loaded {
		// Upload of this same binary is already in progress, we should wait
		// until it's complete and then return
		err := t.WaitForBinaryUploadComplete(binariesInS3)
		if err == nil {
			return binariesInS3, nil
		}
		return "", err
	}

	err = t.awsm.UploadToS3IfNotExists(common.S3Upload{
		SourcePath:   sourcePath,
		TargetRegion: os.Getenv("AWS_REGION"),
		TargetBucket: os.Getenv("BINARIES_S3_BUCKET"),
		TargetPath:   binariesInS3,
	})
	t.pendingBinaryUploads.Delete(binariesInS3)
	if err != nil {
		return "", err
	}
	return binariesInS3, nil
}

// WaitForBinaryUploadComplete will wait until a certain path's upload is either
// not in progress or completed
func (t *TestRunManager) WaitForBinaryUploadComplete(path string) error {
	start := time.Now()
	for time.Now().Before(start.Add(time.Minute * 5)) {
		time.Sleep(time.Second * 1)
		_, stillUploading := t.pendingBinaryUploads.Load(path)
		if !stillUploading {
			return nil
		}
	}
	return errors.New("timeout waiting for upload to complete")
}

// UploadConfig uploads the contents of the configuration file for the system
// to S3 for future reference
func (t *TestRunManager) UploadConfig(cfg []byte, tr *common.TestRun) error {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())

	n, err := file.Write(cfg)
	if n != len(cfg) || err != nil {
		return fmt.Errorf(
			"error writing file. wrote %d of %d, err: %v",
			n,
			len(cfg),
			err,
		)
	}

	path := fmt.Sprintf("testruns/%s/outputs/config.cfg", tr.ID)

	dl := common.S3Download{
		TargetPath:   filepath.Join(common.DataDir(), path),
		SourceRegion: os.Getenv("AWS_REGION"),
		SourceBucket: os.Getenv("BINARIES_S3_BUCKET"),
		SourcePath:   path,
		Retries:      10,
	}

	tr.PendingResultDownloads = append(
		tr.PendingResultDownloads,
		dl)

	return t.awsm.UploadToS3(common.S3Upload{
		SourcePath:   file.Name(),
		TargetRegion: os.Getenv("AWS_REGION"),
		TargetBucket: os.Getenv("BINARIES_S3_BUCKET"),
		TargetPath:   path,
	})
}

// processS3UploadResponse will look at the message and error returned by the
// call to QueryAgentWithTimeout with an UploadFileToS3RequestMsg. It will
// return the error from the function (if any), or check for a mismatching
// message type or false value in the Success member
// TODO: This could probably be generalized to an msg,err,expected type call
func (t *TestRunManager) processS3UploadResponse(
	agentID int32,
	msg wire.Msg,
	err error,
) error {
	if err == nil {
		// Check if the return type is the expected type
		if _, ok := msg.(*wire.UploadFileToS3ResponseMsg); !ok {
			err = fmt.Errorf(
				"expected UploadFileToS3ResponseMsg, got %T",
				msg,
			)
		}
	}
	if err == nil {
		// Check if the return message has true in its Success member
		if resp, ok := msg.(*wire.UploadFileToS3ResponseMsg); ok &&
			!resp.Success {
			err = fmt.Errorf(
				"agent %d returned success=false on S3 upload",
				agentID,
			)
		}
	}
	return err
}
