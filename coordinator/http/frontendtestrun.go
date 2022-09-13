package http

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
)

var frontendRunCache = sync.Map{}

func (h *HttpServer) getFrontendRun(id string) FrontendTestRunListEntry {
	tri, ok := frontendRunCache.Load(id)
	updateCache := false
	var tr FrontendTestRunListEntry
	if !ok {
		updateCache = true
	} else {
		tr = tri.(FrontendTestRunListEntry)
		if tr.Status == common.TestRunStatusRunning || tr.Status == common.TestRunStatusQueued {
			updateCache = true
		}
	}

	if updateCache {
		ftr, _ := h.tr.GetTestRun(id)
		tr = h.makeFrontendRun(ftr)
		frontendRunCache.Store(id, tr)
	}

	return tr
}

func (h *HttpServer) makeFrontendRun(
	tr *common.TestRun,
) FrontendTestRunListEntry {
	var res FrontendTestRunListEntry
	runRoleCounts := []FrontendTestRunRoleCount{}
	for _, r := range tr.Roles {
		added := false
		for i := range runRoleCounts {
			if runRoleCounts[i].RoleType == r.Role {
				runRoleCounts[i].Count = runRoleCounts[i].Count + 1
				added = true
				break
			}
		}
		if !added {
			runRoleCounts = append(
				runRoleCounts,
				FrontendTestRunRoleCount{RoleType: r.Role, Count: 1},
			)
		}
	}
	b, err := json.Marshal(tr)
	if err != nil {
		logging.Warnf("Could not marshal testruns: %v", err)
	}
	err = json.Unmarshal(b, &res)
	if err != nil {
		logging.Warnf("Could not unmarshal testruns: %v", err)
	}
	res.RoleCounts = runRoleCounts
	if tr.Result != nil {
		res.AvgThroughput = tr.Result.ThroughputAvg
		for _, p := range tr.Result.LatencyPercentiles {
			if p.Bucket == 99 {
				res.TailLatency = p.Value
			}
		}
	}
	return res
}

func (h *HttpServer) frontendTestRunList() []FrontendTestRunListEntry {
	var res []FrontendTestRunListEntry
	testRuns := h.tr.GetTestRuns()
	for _, tr := range testRuns {
		if time.Since(tr.Created) < time.Hour*24*90 {
			res = append(res, h.getFrontendRun(tr.ID))
		}
	}
	return res
}

// Limited subset of test run properties used for rendering the list(s)
type FrontendTestRunListEntry struct {
	ID                       string                     `json:"id"`
	CreatedByThumbprint      string                     `json:"createdByuserThumbprint"`
	Created                  time.Time                  `json:"created"`
	Started                  time.Time                  `json:"started"`
	Completed                time.Time                  `json:"completed"`
	DontRunUntil             time.Time                  `json:"notBefore"`
	Status                   common.TestRunStatus       `json:"status"`
	Architecture             string                     `json:"architectureID"`
	SweepID                  string                     `json:"sweepID"`
	SweepOneAtATime          bool                       `json:"sweepOneAtATime"`
	RoleCounts               []FrontendTestRunRoleCount `json:"roleCounts"`
	Details                  string                     `json:"details"`
	AvgThroughput            float64                    `json:"avgThroughput"`
	TailLatency              float64                    `json:"tailLatency"`
	PerformanceDataAvailable bool                       `json:"performanceDataAvailable"`
	InvalidTxRate            float64                    `json:"invalidTxRate"`
	FixedTxRate              float64                    `json:"fixedTxRate"`
	PreseedCount             int64                      `json:"preseedCount"`
	ContentionRate           float64                    `json:"contentionRate"`
	LoadGenTxType            string                     `json:"loadGenTxType"`
	PreseedShards            bool                       `json:"preseedShards"`
	ShardReplicationFactor   int                        `json:"shardReplicationFactor"`
	LoadGenOutputCount       int                        `json:"loadGenOutputCount"`
	LoadGenInputCount        int                        `json:"loadGenInputCount"`
	SentinelAttestations     int                        `json:"sentinelAttestations"`
	AuditInterval            int                        `json:"auditInterval"`
}

type FrontendTestRunRoleCount struct {
	RoleType common.SystemRole `json:"role"`
	Count    int               `json:"count"`
}
