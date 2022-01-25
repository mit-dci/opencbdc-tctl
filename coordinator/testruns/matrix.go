package testruns

import (
	"fmt"
	"time"

	"github.com/mit-dci/cbdc-test-controller/common"
)

// GenerateMatrix generates a matrix of results from all testruns. Note that
// this is a quite costly function
func (t *TestRunManager) GenerateMatrix() ([]*common.MatrixResult, time.Time, time.Time) {
	trs := t.GetTestRuns()
	return t.GenerateMatrixForRuns(trs)
}

// GenerateSweepMatrix generates a matrix of results from all testruns in a
// single sweep
func (t *TestRunManager) GenerateSweepMatrix(
	sweepIDs []string,
) ([]*common.MatrixResult, time.Time, time.Time) {
	trs := t.GetTestRuns()
	sweepRuns := make([]*common.TestRun, 0)
	for i := range trs {
		for _, sweepID := range sweepIDs {
			if trs[i].SweepID == sweepID {
				sweepRuns = append(sweepRuns, trs[i])
			}
		}
	}
	return t.GenerateMatrixForRuns(sweepRuns)
}

// GenerateMatrixForRuns generates a matrix of results for all test runs passed
// in the `trs` parameter. A matrix will identify the different configurations
// in the set of testruns, and average the testresults that share the same
// configuration. The return value is a list of matrix results as well as the
// earliest and latest time tests in this set were executed
func (t *TestRunManager) GenerateMatrixForRuns(
	trs []*common.TestRun,
) ([]*common.MatrixResult, time.Time, time.Time) {
	configs := map[string]*common.TestRunNormalizedConfig{}
	results := map[string][]*common.TestResult{}

	// Keep track of the first and last test run that is part of this set
	minDate := time.Date(4000, time.January, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC)

	// Bucket the results from all test runs using the hash of their
	// TestRunNormalizedConfig
	for _, tr := range trs {
		if tr.Status == common.TestRunStatusCompleted {
			// Only consider completed test runs
			if tr.Completed.After(maxDate) {
				maxDate = tr.Completed
			}
			if tr.Completed.Before(minDate) {
				minDate = tr.Completed
			}
			dummy := false
			if tr.Result == nil {
				dummy = true
				tr.Result = &common.TestResult{
					LatencyPercentiles: []common.TestResultPercentile{
						{Bucket: 99.999, Value: 0},
					},
					ThroughputPercentiles: []common.TestResultPercentile{
						{Bucket: 99.999, Value: 0},
					},
				}
			}
			cfg := tr.NormalizedConfig()
			cfgHash := fmt.Sprintf("%x", cfg.Hash())
			arr, exists := results[cfgHash]
			if !exists {
				arr = []*common.TestResult{}
				configs[cfgHash] = cfg
			}
			arr = append(arr, tr.Result)
			results[cfgHash] = arr
			if dummy {
				tr.Result = nil
			}
		}
	}

	// For each different configuration, add a row to the result matrix
	res := make([]*common.MatrixResult, 0)
	for k, v := range configs {
		// Append the matrixresult which includes the configuration that all
		// results share, a testresult that will serve as a summary, including
		// the min/max and averages over all test runs, as well as all
		// individual test run results.
		mr := &common.MatrixResult{
			Config:      v,
			ResultAvg:   &common.TestResult{},
			ResultCount: len(results[k]),
			Results:     results[k],
		}
		for _, r := range results[k] {
			// Summarize the results - if the testrun's min or max exceeds the
			// summary's min or max, update it. For average, stddev and the
			// percentile buckets, add this testrun's value to the summary's
			// value - at the end we divide the summary's value for those
			// properties by the number of test runs to get the mean of these
			// values
			mr.ResultAvg.LatencyAvg += r.LatencyAvg
			if mr.ResultAvg.LatencyMax < r.LatencyMax {
				mr.ResultAvg.LatencyMax = r.LatencyMax
			}
			if mr.ResultAvg.LatencyMin > r.LatencyMin {
				mr.ResultAvg.LatencyMin = r.LatencyMin
			}
			mr.ResultAvg.LatencyStd += r.LatencyStd

			mr.ResultAvg.ThroughputAvg += r.ThroughputAvg
			mr.ResultAvg.ThroughputAvg2 += r.ThroughputAvg2
			if mr.ResultAvg.ThroughputMax < r.ThroughputMax {
				mr.ResultAvg.ThroughputMax = r.ThroughputMax
			}
			if mr.ResultAvg.ThroughputMin > r.ThroughputMin {
				mr.ResultAvg.ThroughputMin = r.ThroughputMin
			}
			mr.ResultAvg.ThroughputStd += r.ThroughputStd

			for i := range r.LatencyPercentiles {
				pctFound := false
				for j := range mr.ResultAvg.LatencyPercentiles {
					if mr.ResultAvg.LatencyPercentiles[j].Bucket == r.LatencyPercentiles[i].Bucket {
						mr.ResultAvg.LatencyPercentiles[j] = common.TestResultPercentile{
							Bucket: mr.ResultAvg.LatencyPercentiles[j].Bucket,
							Value:  mr.ResultAvg.LatencyPercentiles[j].Value + r.LatencyPercentiles[i].Value,
						}
						pctFound = true
					}
				}
				if !pctFound {
					mr.ResultAvg.LatencyPercentiles = append(
						mr.ResultAvg.LatencyPercentiles,
						r.LatencyPercentiles[i],
					)
				}
			}

			for i := range r.ThroughputPercentiles {
				pctFound := false
				for j := range mr.ResultAvg.ThroughputPercentiles {
					if mr.ResultAvg.ThroughputPercentiles[j].Bucket == r.ThroughputPercentiles[i].Bucket {
						mr.ResultAvg.ThroughputPercentiles[j] = common.TestResultPercentile{
							Bucket: mr.ResultAvg.ThroughputPercentiles[j].Bucket,
							Value:  mr.ResultAvg.ThroughputPercentiles[j].Value + r.ThroughputPercentiles[i].Value,
						}
						pctFound = true
					}
				}
				if !pctFound {
					mr.ResultAvg.ThroughputPercentiles = append(
						mr.ResultAvg.ThroughputPercentiles,
						r.ThroughputPercentiles[i],
					)
				}
			}
		}

		// Get the mean value of these properties in the result summary
		mr.ResultAvg.LatencyAvg = mr.ResultAvg.LatencyAvg / float64(
			mr.ResultCount,
		)
		mr.ResultAvg.LatencyStd = mr.ResultAvg.LatencyStd / float64(
			mr.ResultCount,
		)
		mr.ResultAvg.ThroughputAvg = mr.ResultAvg.ThroughputAvg / float64(
			mr.ResultCount,
		)
		mr.ResultAvg.ThroughputAvg2 = mr.ResultAvg.ThroughputAvg2 / float64(
			mr.ResultCount,
		)
		mr.ResultAvg.ThroughputStd = mr.ResultAvg.ThroughputStd / float64(
			mr.ResultCount,
		)
		for j := range mr.ResultAvg.ThroughputPercentiles {
			mr.ResultAvg.ThroughputPercentiles[j] = common.TestResultPercentile{
				Bucket: mr.ResultAvg.ThroughputPercentiles[j].Bucket,
				Value: mr.ResultAvg.ThroughputPercentiles[j].Value / float64(
					mr.ResultCount,
				),
			}
		}
		for j := range mr.ResultAvg.LatencyPercentiles {
			mr.ResultAvg.LatencyPercentiles[j] = common.TestResultPercentile{
				Bucket: mr.ResultAvg.LatencyPercentiles[j].Bucket,
				Value: mr.ResultAvg.LatencyPercentiles[j].Value / float64(
					mr.ResultCount,
				),
			}
		}
		res = append(res, mr)
	}

	return res, minDate, maxDate
}
