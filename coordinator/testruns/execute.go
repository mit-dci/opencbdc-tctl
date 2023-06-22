package testruns

import (
	"errors"
	"fmt"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
)

// ExecuteTestRun is the main function that executes a test run
func (t *TestRunManager) ExecuteTestRun(tr *common.TestRun) {
	// Persist the commithash of the controller on which the test executed. This
	// could be helpful in determining why two testruns yield different results
	// if all other parameters are equal - meaning a change in the test
	// controller's behaviour has caused the difference and knowing the commit
	// hashes of the test controller at the time of execution could be helpful
	// in spotting that.
	tr.ControllerCommit = t.commitHash
	t.PersistTestRun(tr)

	errs := t.ValidateTestRun(tr)
	if len(errs) > 0 {
		for _, err := range errs {
			t.WriteLog(tr, "Error in test run configuration: %v", err)
		}
		t.FailTestRun(tr,
			fmt.Errorf("%d error(s) in the test run configuration "+
				" - see test run log for details", len(errs)))
		return
	}

	var binariesInS3 string
	binariesInS3, err := t.BinariesExistInS3(tr, false)
	if err != nil {
		t.FailTestRun(
			tr,
			fmt.Errorf("Checking binary existence failed: %v", err),
		)
		return
	}
	if binariesInS3 == "" {
		err := t.CompileBinaries(tr, false)
		if err != nil {
			t.FailTestRun(tr, fmt.Errorf("Compilation failed: %v", err))
			return
		}

		binariesInS3, err = t.UploadBinaries(tr, false)
		if err != nil {
			t.FailTestRun(
				tr,
				fmt.Errorf("Failed to upload binaries to S3: %v", err),
			)
			return
		}
	}

	tr.SeederHash, err = t.src.FindMostRecentCommitChangingSeeder(tr.CommitHash)
	if err != nil {
		t.FailTestRun(tr, fmt.Errorf("Failed determining seeder hash: %v", err))
	}
	seederBinariesInS3, err := t.BinariesExistInS3(tr, true)
	if err != nil {
		t.FailTestRun(
			tr,
			fmt.Errorf("Checking seeder binary existence failed: %v", err),
		)
		return
	}
	if seederBinariesInS3 == "" {
		err = t.CompileBinaries(tr, true)
		if err != nil {
			t.FailTestRun(tr, fmt.Errorf("Seeder compilation failed: %v", err))
			return
		}

		_, err = t.UploadBinaries(tr, true)
		if err != nil {
			t.FailTestRun(
				tr,
				fmt.Errorf("Failed to upload seeder binaries to S3: %v", err),
			)
			return
		}
	}

	if !t.IsParsec(tr.Architecture) {
		// Generate the configuration file the system needs based on the
		// configured
		// parameters in the UI
		dummyCfg, err := t.GenerateConfig(tr, true)
		if err != nil {
			t.FailTestRun(tr, err)
			return
		}

		err = t.CheckPreseed(tr, dummyCfg)
		if err != nil {
			if err != errAbortedWhilePreseeding {
				t.FailTestRun(tr, fmt.Errorf("Preseeding failed: %v", err))
			}
			return
		}
	}

	if t.HasAWSRoles(tr) {
		// Spawn AWS Agents
		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			"Spawning agents in AWS",
		)

		failed, timeout, terminated := false, false, false
		if !t.SpawnAWSInstances(tr) {
			failed = true
		} else {
			failed, timeout, terminated = t.WaitForAWSInstances(tr)
		}

		if failed {
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				"Failed to launch agents - stopping the ones that launched",
			)
		} else if terminated {
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				"Test run terminated - stopping AWS agents",
			)
		} else if timeout {
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				"AWS agents took too long to come online - stopping them",
			)
		}

		if failed || timeout || terminated {
			// Kill all spawned instances
			err := t.KillAwsAgents(tr)
			if err != nil {
				t.FailTestRun(
					tr,
					errors.New(
						"Timed out / terminated / failed "+
							"but could not kill AWS agents, kill manually",
					),
				)
			} else {
				if terminated {
					t.UpdateStatus(
						tr,
						common.TestRunStatusAborted,
						"Test run terminated manually")
				} else if failed {
					t.UpdateStatus(
						tr,
						common.TestRunStatusFailed,
						"AWS agents failed to launch")
				} else if timeout {
					t.UpdateStatus(
						tr,
						common.TestRunStatusFailed,
						"AWS agents took too long to come online")
				}
			}
			return
		}
	}

	// Make a channel to receive completion of commands
	cmd := make(chan *common.ExecutedCommand, 10)

	// Make a channel to send commands that exited with a non-zero exit code
	failures := make(chan *common.ExecutedCommand, 10)

	// Start a goroutine to read from cmd, interpret, and send to failures if
	// needed
	go func() {
		for c := range cmd {
			if c.ExitCode != 0 {
				// Deliberate failures are commands that we failed because the
				// testrun defined these commands to be terminated during the
				// course of the testrun. We shouldn't treat these as failures
				// because they would cause the entire test to abort.
				deliberate := false
				for _, id := range tr.DeliberateFailures {
					if id == c.CommandID {
						deliberate = true
					}
				}
				if !deliberate {
					// For non-deliberate failures, we send the command with
					// the non-zero exit code to the failures channel. We listen
					// on this channel in RunBinaries and exit that function
					// when a failure occurred
					select {
					case failures <- c:
					default:
					}
				}
			}
			// If the exit code is zero, this was a succesfully executed command
			// so we should add it to the commands the testrun executed.
			tr.AddExecutedCommand(c)
		}
	}()

	// Snapshot the agents so we know the exact system information of the agents
	// on which we run the test before actually doing anything
	t.SnapshotAgents(tr)

	// Create environment folders on each agent and deploy the binaries into
	// them
	var envs map[int32][]byte
	envs, err = t.DeployBinaries(tr, binariesInS3)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	// Generate the configuration file the system needs based on the configured
	// parameters in the UI
	cfg, err := t.GenerateConfig(tr, false)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	t.WriteLog(tr, "Test run config:\n%s", string(cfg))

	// Upload the config to S3 for persistence - this was done from all of the
	// roles previously but the file is the same for all so just upload it once
	err = t.UploadConfig(cfg, tr)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	// Write the configuration file to all agents, placing it in the environment
	// folders created by DeployBinaries above
	err = t.DeployConfig(tr, envs, cfg)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	// Instruct the agents that will run the shards to download the preseed data
	// for the shards from S3
	err = t.PreseedShards(tr, envs)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	// Call RunBinaries to actually start up all the system components on the
	// agents and conduct the actual test. At the end of a succeeded or failed
	// test run, RunBinaries will also instruct the agents to upload all their
	// performance profiling data to S3 and update the `PendingResultDownloads`
	// member of the test run with all of the performance profiles available for
	// download
	err = t.RunBinaries(tr, envs, cmd, failures)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	// RunBinaries can call FailTestRun and not return the error to this
	// routine, in which case we should not continue
	if tr.Status != common.TestRunStatusRunning {
		return
	}

	// Instruct the agents to upload all their outputs to S3 and update the
	// `PendingResultDownloads` member of the test run with all of the output
	// files available for download.
	err = t.CopyOutputs(tr, envs, false)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	// Now that the agents have copied all of their test result files and
	// performance profiles to S3 we can safely kill all the agents without
	// losing anything valuable
	var killErr error
	if t.HasAWSRoles(tr) {
		t.UpdateStatus(
			tr,
			common.TestRunStatusRunning,
			"Run complete, killing spawned AWS agents",
		)
		killErr = t.KillAwsAgents(tr)
	}

	// Signal the instances are stopped - this is used by the scheduling logic
	// to no longer count the roles in this test run against the parallel agent
	// limit (or VCPU limits), which frees up the space to start new tests
	tr.AWSInstancesStopped = true

	// Download all files that our agents copied to S3
	t.WriteLog(
		tr,
		"Downloading %d outputs and performance profiles from S3",
		len(tr.PendingResultDownloads),
	)
	err = t.awsm.DownloadMultipleFromS3(tr.PendingResultDownloads)
	if err != nil {
		t.FailTestRun(tr, err)
		return
	}

	// Calculate the test results
	t.UpdateStatus(tr, common.TestRunStatusRunning, "Calculating test results")
	_, err = t.CalculateResults(tr, false)
	if err != nil {
		t.WriteLog(tr, "Test result calculation failed: %v", err)
	}

	if killErr != nil {
		t.UpdateStatus(
			tr,
			common.TestRunStatusCompleted,
			"Completed, but was unable to kill AWS agent(s)",
		)
		return
	}

	// Done!
	t.UpdateStatus(tr, common.TestRunStatusCompleted, "Completed")

	// Complete - run the next run of the sweep if doing a one-at-a-time sweep
	if tr.SweepOneAtATime {
		t.ContinueSweep(tr, tr.SweepID)
	}
}

func (t *TestRunManager) CleanupCommands(
	tr *common.TestRun,
	allCmds []runningCommand,
	envs map[int32][]byte,
) error {
	// Break all commands that are still running - if one command fails or only
	// the archiver has succesfully completed, we still need to terminate all
	// the other commands using a interrupt or kill signal. This would trigger
	// the finishing of all stdout/err buffers and terminating any performance
	// profiling running alongside the commands
	err := t.BreakAndTerminateAllCmds(tr, allCmds)
	if err != nil {
		return err
	}

	// Time for the commands to break and commit perf results
	time.Sleep(time.Second * 5)

	// Trigger the agents to upload the performance data for all commands
	// to S3
	err = t.GetPerformanceProfiles(tr, allCmds, envs)
	if err != nil {
		return err
	}

	err = t.GetLogFiles(tr, allCmds, envs)
	if err != nil {
		return err
	}
	return nil
}
