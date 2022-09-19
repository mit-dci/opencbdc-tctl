package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/mit-dci/opencbdc-tctl/logging"
)

func FindMissingSweepRuns(trs []*TestRun, sweepID string) []*TestRun {
	sweepRuns := make([]*TestRun, 0)
	succeededSweepRuns := make([]*TestRun, 0)
	for i := range trs {
		if trs[i].SweepID == sweepID {
			sweepRuns = append(sweepRuns, trs[i])
			if trs[i].Status == TestRunStatusCompleted ||
				trs[i].Status == TestRunStatusQueued ||
				trs[i].Status == TestRunStatusRunning {
				succeededSweepRuns = append(succeededSweepRuns, trs[i])
			}
		}
	}

	if sweepRuns[0].Sweep == "peak" {
		// Peak finding sweeps work a little differently
		if len(succeededSweepRuns) == 0 {
			return []*TestRun{} // No way to determine sweep
		} else if len(succeededSweepRuns) == 2 {
			// finished initial peak finding, schedule confirmation runs
			runs, err := GetConfirmationPeakFindingRuns(succeededSweepRuns)
			if err != nil {
				logging.Errorf("Error calculating next peak finding runs: %v", err)
				return []*TestRun{}
			}
			return runs
		} else if len(succeededSweepRuns) > 2 {
			// Done!
			return []*TestRun{}
		}

		tr, err := GetNextPeakFindingRun(succeededSweepRuns)
		if err != nil || tr == nil {
			logging.Errorf("Error calculating next peak finding run: %v", err)
			return []*TestRun{}
		}
		return []*TestRun{tr}
	}

	var firstRun *TestRun
	for _, runs := range [][]*TestRun{succeededSweepRuns, sweepRuns} {
		for i := range runs {
			if firstRun == nil {
				firstRun = runs[i]
			}
			if runs[i].Created.Before(firstRun.Created) {
				firstRun = runs[i]
			}
		}
		if firstRun != nil {
			break
		}
	}

	if firstRun == nil {
		return []*TestRun{} // No way to determine sweep
	}

	expectedRuns := ExpandSweepRun(firstRun, firstRun.SweepID)

	logging.Infof(
		"Sweep ID has %d expected runs, checking which ones are there in %d completed/queued/running sweep runs",
		len(expectedRuns),
		len(succeededSweepRuns),
	)
	for i := 0; i < len(expectedRuns); i++ {
		found := false
		removeSweepRun := -1

		expNC := expectedRuns[i].NormalizedConfigWithAgentData(false)
		expNC.ControllerCommitHash = ""
		expectedRunHash := fmt.Sprintf("%x", expNC.Hash())
		for j := range succeededSweepRuns {
			sweepNC := succeededSweepRuns[j].NormalizedConfigWithAgentData(
				false,
			)
			sweepNC.ControllerCommitHash = ""
			sweepRunHash := fmt.Sprintf("%x", sweepNC.Hash())
			if sweepRunHash == expectedRunHash {
				found = true
				removeSweepRun = j
				break
			}
		}

		if found {
			logging.Infof(
				"Sweep run %d matches expected run %d",
				removeSweepRun,
				i,
			)
			succeededSweepRuns[len(succeededSweepRuns)-1], succeededSweepRuns[removeSweepRun] = succeededSweepRuns[removeSweepRun], succeededSweepRuns[len(succeededSweepRuns)-1]
			succeededSweepRuns = succeededSweepRuns[:len(succeededSweepRuns)-1]
			expectedRuns = append(expectedRuns[:i], expectedRuns[i+1:]...)
			i--
		}
	}

	return expectedRuns
}

func GetConfirmationPeakFindingRuns(succeededSweepRuns []*TestRun) ([]*TestRun, error) {
	runs := make([]*TestRun, 0)

	sort.SliceStable(succeededSweepRuns, func(i, j int) bool {
		return succeededSweepRuns[i].Created.Before(succeededSweepRuns[j].Created)
	})

	// Get the last run's bandwidth
	baseRun := succeededSweepRuns[len(succeededSweepRuns)-1]
	if baseRun.Result.ThroughputPeakLB == 0 || baseRun.Result.ThroughputPeakUB == 0 {
		return nil, fmt.Errorf("base run has no peak lower/upper bound - cannot continue")
	}

	// Get average between UB and LB, take -5% and +5% for confirmation levels, round to nearest 500 tps increment
	avg := (baseRun.Result.ThroughputPeakLB + baseRun.Result.ThroughputPeakUB) / 2
	confirmPeak := int(math.Floor(avg*0.95/500) * 500)
	confirmAbovePeak := int(math.Floor(avg*1.05/500) * 500)

	// Create 3 test runs for peak and above peak levels
	for _, tps := range []int{confirmPeak, confirmAbovePeak} {
		for i := 0; i < 3; i++ {
			buf, _, err := GetTestRunCopy(baseRun)
			if err != nil {
				return nil, fmt.Errorf("error getting testrun copy: %v", err)
			}
			var newTr TestRun
			err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&newTr)
			if err != nil {
				return nil, fmt.Errorf("error deserializing testrun: %v", err)
			}
			newTr.SweepID = baseRun.SweepID
			newTr.LoadGenTPSTarget = tps
			newTr.LoadGenTPSStepStart = 1
			newTr.LoadGenTPSStepPercent = 0
			newTr.LoadGenTPSStepTime = 0
			runs = append(runs, &newTr)
		}
	}
	return runs, nil
}

func GetNextPeakFindingRun(succeededSweepRuns []*TestRun) (*TestRun, error) {
	sort.SliceStable(succeededSweepRuns, func(i, j int) bool {
		return succeededSweepRuns[i].Created.Before(succeededSweepRuns[j].Created)
	})

	// Get the last one and "zoom in"
	baseRun := succeededSweepRuns[len(succeededSweepRuns)-1]
	buf, _, err := GetTestRunCopy(baseRun)
	if err != nil {
		return nil, fmt.Errorf("error getting testrun copy: %v", err)
	}

	var newTr TestRun
	err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&newTr)
	if err != nil {
		return nil, fmt.Errorf("error deserializing testrun: %v", err)
	}
	newTr.SweepID = baseRun.SweepID

	if baseRun.Result.ThroughputPeakLB == 0 || baseRun.Result.ThroughputPeakUB == 0 {
		return nil, fmt.Errorf("base run has no peak lower/upper bound - cannot continue")
	}

	newTr.LoadGenTPSTarget = int(baseRun.Result.ThroughputPeakUB)
	newTr.LoadGenTPSStepStart = baseRun.Result.ThroughputPeakLB / baseRun.Result.ThroughputPeakUB
	newTr.LoadGenTPSStepPercent = -1
	newTr.LoadGenTPSStepTime = 20

	return &newTr, nil
}

func GetTestRunCopy(originalTr *TestRun) (bytes.Buffer, *TestRun, error) {
	var tr TestRun
	var buf bytes.Buffer
	var err error
	err = json.NewEncoder(&buf).Encode(originalTr)
	if err != nil {
		return buf, &tr, fmt.Errorf("error serializing testrun: %v", err)
	}
	err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&tr)
	if err != nil {
		return buf, &tr, fmt.Errorf("error deserializing testrun: %v", err)
	}

	for i := range tr.Roles {
		tr.Roles[i].AgentID = -1
	}

	buf.Reset()
	err = json.NewEncoder(&buf).Encode(&tr)
	if err != nil {
		return buf, &tr, fmt.Errorf("error serializing testrun: %v", err)
	}
	return buf, &tr, nil
}

func ExpandSweepRun(originalTr *TestRun, sweepID string) []*TestRun {
	runs := make([]*TestRun, 0)
	buf, tr, err := GetTestRunCopy(originalTr)
	if err != nil {
		logging.Errorf("error getting testrun copy: %v", err)
		return runs
	}
	if tr.Sweep == "" {
		if tr.Repeat == 1 {
			// No sweep, just run
			runs = append(runs, originalTr)
			return runs
		} else {
			for repeat := 0; repeat < tr.Repeat; repeat++ {
				var newTr TestRun
				err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&newTr)
				if err != nil {
					logging.Errorf("Error deserializing testrun: %v", err)
				}
				newTr.SweepID = sweepID
				runs = append(runs, &newTr)
			}
		}
	} else if tr.Sweep == "peak" {
		// Assign sweep ID but just run one, peak == one at a time
		var newTr TestRun
		err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&newTr)
		if err != nil {
			logging.Errorf("Error deserializing testrun: %v", err)
		}
		newTr.SweepID = sweepID

		// Initial peak finding runs should use sufficient steps and duration
		// TODO: Atomizer account for block time
		newTr.SampleCount = 600
		newTr.LoadGenTPSStepPercent = 0.01
		newTr.LoadGenTPSStepTime = 5
		runs = append(runs, &newTr)
		return runs
	} else if tr.Sweep == "parameter" {
		var raw map[string]interface{}
		err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&raw)
		if err != nil {
			logging.Errorf("Error deserializing testrun: %v", err)
		}

		for i := tr.SweepParameterStart; i <= tr.SweepParameterStop; i += tr.SweepParameterIncrement {
			for repeat := 0; repeat < tr.Repeat; repeat++ {
				raw[tr.SweepParameter] = i
				b, err := json.Marshal(raw)
				if err != nil {
					logging.Errorf("Error serializing testrun: %v", err)
				}

				var sweeptr TestRun
				err = json.Unmarshal(b, &sweeptr)
				if err != nil {
					logging.Errorf("Error deserializing testrun: %v", err)
				}

				sweeptr.SweepID = sweepID
				runs = append(runs, &sweeptr)
			}
		}
	} else if tr.Sweep == "time" {
		for repeat := 0; repeat < tr.Repeat; repeat++ {
			for i := 0; i < tr.SweepTimeRuns; i++ {
				var sweeptr TestRun
				err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&sweeptr)
				if err != nil {
					logging.Errorf("Error deserializing testrun: %v", err)
				}

				sweeptr.DontRunBefore = time.Now().Add(time.Duration(i) * time.Duration(sweeptr.SweepTimeMinutes) * time.Minute)
				sweeptr.SweepID = sweepID
				runs = append(runs, &sweeptr)
			}
		}
	} else if tr.Sweep == "roles" {
		for i := 0; i < tr.SweepRoleRuns; i++ {
			for repeat := 0; repeat < tr.Repeat; repeat++ {
				var sweeptr TestRun
				err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&sweeptr)
				if err != nil {
					logging.Errorf("Error deserializing testrun: %v", err)
				}

				// Get Role Counts
				roleCounts := map[SystemRole]int{}
				for _, r := range sweeptr.Roles {
					c, ok := roleCounts[r.Role]
					if ok {
						roleCounts[r.Role] = c + 1
					} else {
						roleCounts[r.Role] = 1
					}
				}
				// Add the roles
				for j := 0; j < i; j++ {
					for _, r := range sweeptr.SweepRoles {
						c, ok := roleCounts[r.Role]
						if !ok {
							c = 0
						}
						sweeptr.Roles = append(sweeptr.Roles, &TestRunRole{
							Role:                r.Role,
							Index:               c,
							AwsLaunchTemplateID: r.AwsLaunchTemplateID,
							AgentID:             -1,
						})
						roleCounts[r.Role] = c + 1
					}
				}
				sweeptr.SweepID = sweepID
				runs = append(runs, &sweeptr)
			}
		}
	}
	return runs
}
