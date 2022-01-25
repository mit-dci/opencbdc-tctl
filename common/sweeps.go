package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mit-dci/cbdc-test-controller/logging"
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

func ExpandSweepRun(originalTr *TestRun, sweepID string) []*TestRun {
	var tr TestRun
	var buf bytes.Buffer
	var err error
	err = json.NewEncoder(&buf).Encode(originalTr)
	if err != nil {
		logging.Errorf("Error serializing testrun: %v", err)
	}
	err = json.NewDecoder(bytes.NewReader(buf.Bytes())).Decode(&tr)
	if err != nil {
		logging.Errorf("Error deserializing testrun: %v", err)
	}

	for i := range tr.Roles {
		tr.Roles[i].AgentID = -1
	}

	buf.Reset()
	err = json.NewEncoder(&buf).Encode(&tr)
	if err != nil {
		logging.Errorf("Error serializing testrun: %v", err)
	}

	runs := make([]*TestRun, 0)
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
