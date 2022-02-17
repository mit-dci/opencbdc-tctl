package testruns

import "github.com/mit-dci/opencbdc-tctl/common"

// SnapshotAgents will take a copy of the current status of the connected agents
// such that we preserve them for later inspection.
func (t *TestRunManager) SnapshotAgents(tr *common.TestRun) {
	agentData := make([]common.TestRunAgentData, 0)
	for _, role := range tr.Roles {
		a, err := t.coord.GetAgent(role.AgentID)
		if err == nil {
			agentData = append(agentData, common.TestRunAgentData{
				AgentID:      a.ID,
				SystemInfo:   a.SystemInfo,
				AgentVersion: a.AgentVersion,
				PingRTT:      a.PingRTT,
				AwsRegion: t.awsm.GetLaunchTemplateRegion(
					role.AwsLaunchTemplateID,
				),
			})
		}
	}

	tr.AgentDataAtStart = agentData
}
