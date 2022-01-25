package testruns

import (
	"sort"

	"github.com/mit-dci/cbdc-test-controller/common"
)

// ContinueSweep will identify the next test run in a one-at-a-time test sweep
// and schedule it for execution
func (t *TestRunManager) ContinueSweep(tr *common.TestRun, sweepID string) {
	if sweepID == "" && tr != nil {
		sweepID = tr.SweepID
	}

	// Get all test runs that are part of the sweep
	sweepRuns := []*common.TestRun{}
	for _, atr := range t.GetTestRuns() {
		if atr.SweepID == sweepID {
			sweepRuns = append(sweepRuns, atr)
		}
	}

	// If last three runs have > 10s latency, stop the sweep
	stopSweep := false
	if len(sweepRuns) >= 3 {
		// Sort the test runs by creation datetime
		sort.Slice(sweepRuns, func(i, j int) bool {
			return sweepRuns[i].Created.After(sweepRuns[j].Created)
		})

		stopSweep = true
		for i := 0; i < 3; i++ {
			if sweepRuns[i].Result != nil {
				for _, pct := range sweepRuns[i].Result.LatencyPercentiles {
					if pct.Bucket == 99 && pct.Value < 10 {
						t.WriteLog(
							tr,
							"Found run %s below 10s latency - not stopping sweep",
							sweepRuns[i].ID,
						)
						stopSweep = false
						break
					}
				}
				if !stopSweep {
					break
				}
			}
		}
	}

	if !stopSweep {
		// If the last three runs have an average throughput lower than the
		// six runs before, then we should also stop sweeping. More client
		// load leading to lower throughput means the system is getting over-
		// loaded
		if len(sweepRuns) >= 9 {
			// Sort sweeps newest to oldest
			sort.Slice(sweepRuns, func(i, j int) bool {
				return sweepRuns[i].Created.After(sweepRuns[j].Created)
			})

			lastThreeAvg := float64(0)
			sixBeforeAvg := float64(0)
			results := 0
			i := 0
			for i = 0; i < len(sweepRuns) && results < 3; i++ {
				if sweepRuns[i].Result != nil {
					lastThreeAvg += sweepRuns[i].Result.ThroughputAvg
					results++
				}
			}
			if results == 3 {
				results = 0
				for ; i < len(sweepRuns) && results < 6; i++ {
					if sweepRuns[i].Result != nil {
						sixBeforeAvg += sweepRuns[i].Result.ThroughputAvg
					}
					results++
				}
				if results == 6 {
					lastThreeAvg /= 3
					sixBeforeAvg /= 6

					if sixBeforeAvg > lastThreeAvg {
						t.WriteLog(
							tr,
							"Last three runs had lower throughput (%.0f) than the six runs before it (%.0f) - stopping sweep",
							lastThreeAvg,
							sixBeforeAvg,
						)
						stopSweep = true
					}
				} else {
					t.WriteLog(tr, "Less than nine results, cannot check throughput degradation - not stopping sweep")
				}
			} else {
				t.WriteLog(tr, "Less than three results, cannot check throughput degradation - not stopping sweep")
			}
		}
	} else {
		t.WriteLog(tr, "Last three runs all had a latency above 10s - stopping sweep")
	}

	if !stopSweep {
		t.WriteLog(tr, "Scheduling next sweep run")
		missing := common.FindMissingSweepRuns(t.GetTestRuns(), tr.SweepID)
		if len(missing) > 0 {
			missing[0].AWSInstancesStopped = false
			for j := range missing[0].Roles {
				missing[0].Roles[j].AgentID = -1
			}
			missing[0].Result = nil
			t.ScheduleTestRun(missing[0])
		}
	}
}
