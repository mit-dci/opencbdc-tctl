package testruns

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/coordinator"
	"github.com/mit-dci/cbdc-test-controller/logging"
)

// PersistTestRun stores the test run data in the persisted state. At present,
// this is a flat directory structure with JSON files. This could be changed
// into a database at some point - but this has been deemed not a priority.
func (t *TestRunManager) PersistTestRun(tr *common.TestRun) {
	// We store test results in a separate file, such that we can have it
	// calculated by a python script separate from the main test controller
	// code. We do keep the results in memory in a property of the test run,
	// but we do not want to persist this in the metadata.
	// We cache the result in a local variable such that we can restore it
	// after saving
	res := tr.Result
	tr.Result = nil

	// Determine the path at which to store the test run metadata. This is
	// a file `metadata.json` inside a folder per test run named by its ID
	testRunDir := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s", tr.ID),
	)
	err := os.MkdirAll(testRunDir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		logging.Errorf("Error creating testrun dir: %v", err)
	}
	f, err := os.OpenFile(
		filepath.Join(testRunDir, "metadata.json"),
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0644,
	)
	if err != nil {
		logging.Warnf("Unable to persist testrun %s: %v", tr.ID, err)
		return
	}
	defer f.Close()

	// Encode the testrun as JSON into the created file
	err = json.NewEncoder(f).Encode(tr)
	if err != nil {
		logging.Warnf("Unable to persist testrun %s: %v", tr.ID, err)
		return
	}

	// Restore cached value
	tr.Result = res
}

// LoadTestResult loads the test result, which is stored separately from the
// test run's metadata, from disk
func (t *TestRunManager) LoadTestResult(tr *common.TestRun) {
	// Load test results from disk if available
	testRunDir := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s", tr.ID),
	)
	f, err := os.OpenFile(
		filepath.Join(
			testRunDir,
			fmt.Sprintf("results%d.json", TestResultVersion),
		),
		os.O_RDONLY,
		0644,
	)
	if err == nil {
		defer f.Close()
		var tres common.TestResult
		err = json.NewDecoder(f).Decode(&tres)
		if err != nil {
			logging.Warnf("Unable to decode testrun results %s: %v", tr.ID, err)
		} else {
			tr.Result = &tres
			logging.Debugf("Loaded test result for run %s (%f tps)", tr.Result.ThroughputAvg)
		}
	} else {
		logging.Warnf("Unable to load test results %s: %v", tr.ID, err)
	}
}

// LoadAllTestRuns is ran on start up of the controller to scan the entire
// directory of testrun data and load the relevant test runs into memory
func (t *TestRunManager) LoadAllTestRuns() {
	activeDir := filepath.Join(common.DataDir(), "testruns")
	archiveDir := filepath.Join(activeDir, "archive")
	// Path to store old testruns that failed, such that we're not loading them
	// again and again on startup
	err := os.MkdirAll(archiveDir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		logging.Errorf("Error creating archive dir: %v", err)
	}

	// Keep track of test runs that were interrupted such that we can reschedule
	// them if necessary
	runsToReschedule := make([]*common.TestRun, 0)
	t.testRunsLock.Lock()
	err = filepath.WalkDir(
		activeDir,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				logging.Warnf("Error scanning test run directory: %v", err)
				// If there was an error reading the directory, just quit
				return nil
			}
			relPath, err := filepath.Rel(activeDir, path)
			if err != nil {
				logging.Warnf("Error determining relative path: %v", err)
				return nil
			}
			//logging.Infof("Walkdir: %s - Rel: %s", path, relPath)

			if relPath == "." {
				// Ignore this
				return nil
			}

			if len(relPath) == 12 { // Test runs have a random 6 byte hex ID
				if _, err := hex.DecodeString(relPath); err == nil {
					// This must likely be a valid testrun, try loading it
					testRunID := relPath
					tr, err := t.LoadTestRun(testRunID)
					if err != nil {
						logging.Warnf(
							"Error loading testrun %s: %v",
							testRunID,
							err,
						)
						return nil
					}

					// If test run is "Running" then the coordinator crashed
					// while running it - we should change the state to
					// "interrupted" to prevent the system from trying to resume
					// it but also reschedule it due to failure
					if tr.Status == common.TestRunStatusRunning {
						t.UpdateStatus(
							tr,
							common.TestRunStatusInterrupted,
							"Interrupted",
						)
						t.PersistTestRun(tr)
						if tr.RetryOnFailure {
							runsToReschedule = append(runsToReschedule, tr)
						}
					}

					// Don't load failed run older than a week - they're no
					// longer interesting and do take up memory space
					skip := false
					if tr.Status != common.TestRunStatusCompleted {
						if tr.Created.Before(time.Now().Add(-24 * time.Hour)) {
							skip = true
						}
					}

					if !skip {
						t.testRuns = append(t.testRuns, tr)
					} else {
						// Move test runs we're not loading to an archive folder
						// such that we skip scanning over them the next time
						// around
						newDir := strings.Replace(path, activeDir, archiveDir, 1)
						logging.Infof("Archiving testrun %s by moving folder [%s] to [%s]", tr.ID, path, newDir)
						err = os.Rename(path, newDir)
						if err != nil {
							logging.Warnf(
								"Error moving old testrun %s: %v",
								testRunID,
								err,
							)
						}
					}
					return fs.SkipDir // No need to crawl this folder any further
				} else {
					return fs.SkipDir // No need to crawl this folder any further
				}
			} else {
				return fs.SkipDir // No need to crawl this folder any further
			}
		},
	)
	if err != nil {
		logging.Errorf("Error while loading testruns: %v", err)
	}
	logging.Infof("Done loading test runs")
	t.testRunsLock.Unlock()

	logging.Infof("Rescheduling %d interrupted runs", len(runsToReschedule))
	for i := range runsToReschedule {
		t.Reschedule(runsToReschedule[i])
	}

	t.testRunsLock.Lock()
	t.loadComplete = true
	t.testRunsLock.Unlock()
	// Signal to the frontend that the system has completed its loading
	// and the client can properly retrieve test run data and commence using
	// the system
	t.ev <- coordinator.Event{
		Type: coordinator.EventTypeSystemStateChange,
	}
}

// LoadTestRun loads a single test run from disk
func (t *TestRunManager) LoadTestRun(id string) (*common.TestRun, error) {
	testRunDir := filepath.Join(
		common.DataDir(),
		fmt.Sprintf("testruns/%s", id),
	)
	f, err := os.OpenFile(
		filepath.Join(testRunDir, "metadata.json"),
		os.O_RDONLY,
		0644,
	)
	if err != nil {
		logging.Warnf("Unable to load testrun %s: %v", id, err)
		return nil, err
	}
	defer f.Close()
	var tr common.TestRun
	err = json.NewDecoder(f).Decode(&tr)
	if err != nil {
		logging.Warnf("Unable to decode testrun %s: %v", id, err)
		return nil, err
	}

	// This section defaults certain parameters if they are not present in the
	// JSON structure. This is because these properties were introduced in later
	// updates, which means these properties are missing from earlier test runs
	// and would default to 0 / false in stead
	raw := map[string]interface{}{}
	_, err = f.Seek(0, 0)
	if err != nil {
		logging.Warnf("Unable to seek to start of stream: %v", id, err)
		return nil, err
	}

	err = json.NewDecoder(f).Decode(&raw)
	if err != nil {
		logging.Warnf("Unable to decode testrun %s: %v", id, err)
		return nil, err
	}
	_, ok := raw["trimSamplesAtStart"]
	if !ok {
		tr.TrimSamplesAtStart = 5 // default if null
	}

	_, ok = raw["trimZeroesAtStart"]
	if !ok {
		tr.TrimZeroesAtStart = true // default if null
	}

	_, ok = raw["trimZeroesAtEnd"]
	if !ok {
		tr.TrimZeroesAtEnd = true // default if null
	}

	// Load tail of the test run log
	tr.ReadLogTail()

	// There was a bug that persisted the result as part of the metadata. Force
	// loading it from results.json
	tr.Result = nil

	// Load test result
	if tr.Status == common.TestRunStatusCompleted {
		t.LoadTestResult(&tr)
		// Recalculate missing results for runs completed in the the past 48
		// hours this allows for test runs that uncover a bug in the result
		// calculation to be recalculated without having to increase the result
		// version triggering a recalculation of all testruns.
		if tr.Result == nil &&
			tr.Completed.After(time.Now().Add(-48*time.Hour)) {
			t.resultCalculationChan <- resultCalculation{
				calculateForRun: &tr,
				responseChan:    nil,
			}
		}
	}

	// If a test run was interrupted (had status Running when loading, meaning
	// the controller crashed while running this test) - but the test run log
	// shows that it was already calculating the test results, we can try
	// calculating the test results here - meaning we can preserve the test run
	// and not have to rerun it
	if tr.Status == common.TestRunStatusInterrupted &&
		strings.Contains(
			tr.LogTail(),
			"Updated status to [Running] [Calculating test results]",
		) {
		go func(crtr *common.TestRun) {
			c := make(chan error, 1)
			t.resultCalculationChan <- resultCalculation{
				calculateForRun: &tr,
				responseChan:    c,
			}
			if <-c == nil {
				t.UpdateStatus(
					crtr,
					common.TestRunStatusCompleted,
					"Finished run that was interrupted while calculating results",
				)
				t.PersistTestRun(crtr)
			}
		}(&tr)
	}

	return &tr, nil
}

// TestRunsLoaded indicates if the system has completed loading the test runs
func (t *TestRunManager) TestRunsLoaded() bool {
	return t.loadComplete
}

// GetTestRun returns a single test run by its ID
func (t *TestRunManager) GetTestRun(runID string) (*common.TestRun, bool) {
	for _, tr := range t.testRuns {
		if tr.ID == runID {
			return tr, true
		}
	}
	return nil, false
}

// GetTestRuns returns all testruns known to the system
func (t *TestRunManager) GetTestRuns() []*common.TestRun {
	return t.testRuns
}
