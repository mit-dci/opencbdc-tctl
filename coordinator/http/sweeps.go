package http

import (
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
)

type SweepData struct {
	ID                      string                   `json:"id"`
	ArchitectureID          string                   `json:"architectureID"`
	RunCount                int                      `json:"runCount"`
	SweepType               string                   `json:"sweepType"`
	SweepParameter          string                   `json:"sweepParameter"`
	FirstRun                time.Time                `json:"firstRun"`
	LastRun                 time.Time                `json:"lastRun"`
	FirstRunData            FrontendTestRunListEntry `json:"firstRunData"`
	FirstRunID              string
	SweepRoleRuns           int                    `json:"sweepRoleRuns"`
	SweepParameterStart     float64                `json:"sweepParameterStart"`
	SweepParameterStop      float64                `json:"sweepParameterStop"`
	SweepParameterIncrement float64                `json:"sweepParameterIncrement"`
	SweepRoles              []*common.TestRunRole  `json:"sweepRoles"`
	CommonParameters        map[string]interface{} `json:"commonParameters"`
}

func (h *HttpServer) listSweeps() []*SweepData {
	runs := h.tr.GetTestRuns()
	sweeps := []*SweepData{}
	for _, r := range runs {
		if r.SweepID != "" && r.Status == common.TestRunStatusCompleted {
			found := false
			for i := range sweeps {
				if sweeps[i].ID == r.SweepID {
					sweeps[i].RunCount++
					if sweeps[i].FirstRun.IsZero() ||
						sweeps[i].FirstRun.After(r.Completed) {
						sweeps[i].FirstRun = r.Completed
						sweeps[i].FirstRunID = r.ID
					}
					if sweeps[i].LastRun.IsZero() ||
						sweeps[i].LastRun.Before(r.Completed) {
						sweeps[i].LastRun = r.Completed
					}
					found = true
					break
				}
			}
			if !found {
				sweep := SweepData{
					ID:                      r.SweepID,
					RunCount:                1,
					SweepType:               r.Sweep,
					FirstRun:                r.Completed,
					LastRun:                 r.Completed,
					FirstRunID:              r.ID,
					SweepParameter:          r.SweepParameter,
					SweepRoleRuns:           r.SweepRoleRuns,
					SweepParameterStart:     r.SweepParameterStart,
					SweepParameterStop:      r.SweepParameterStop,
					SweepParameterIncrement: r.SweepParameterIncrement,
					SweepRoles:              r.SweepRoles,
					ArchitectureID:          r.Architecture,
				}
				sweeps = append(sweeps, &sweep)
			}
		}
	}
	for i := range sweeps {
		sweeps[i].FirstRunData = h.getFrontendRun(sweeps[i].FirstRunID)
	}
	return sweeps
}
