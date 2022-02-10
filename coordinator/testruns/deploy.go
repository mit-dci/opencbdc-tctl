package testruns

import (
	"fmt"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/wire"
)

// DeployConfig is a convenience method to send a DeployFileRequestMsg to all
// agents that are part of a testrun with the contents of the system-wide
// configuration file. It will be deployed at a location relative to the
// environment folder in which the test is running on that agent, and its path
// can be resolved through the %CFG% substitution parameter, which we pass to
// all commands we run
func (t *TestRunManager) DeployConfig(
	tr *common.TestRun,
	envs map[int32][]byte,
	cfg []byte,
) error {
	t.UpdateStatus(
		tr,
		common.TestRunStatusRunning,
		"Deploying config to agents (0%)",
	)

	f := func(role *common.TestRunRole) error {
		msg, err := t.am.QueryAgent(role.AgentID, &wire.DeployFileRequestMsg{
			EnvironmentID: envs[role.AgentID],
			File: common.File{
				FilePath: "config.cfg",
				Contents: cfg,
			},
		})
		if err != nil {
			return err
		}
		_, ok := msg.(*wire.DeployFileResponseMsg)
		if !ok {
			return fmt.Errorf("expected DeployFileResponseMsg, got %T", msg)
		}
		return nil
	}
	err := t.RunForAllAgents(f, tr, "Deploying config to agents", time.Minute)
	return err
}

// DeployBinaries deploys the prebuilt binaries to all involved agents, and
// returns a map of agentID => environmentID for all environments created on the
// test agents. It calls PrepareAgentWithBinariesForCommit for each role in the
// testrun
func (t *TestRunManager) DeployBinaries(
	tr *common.TestRun,
	binariesInS3Path string,
) (map[int32][]byte, error) {
	t.UpdateStatus(
		tr,
		common.TestRunStatusRunning,
		"Deploying binaries to agents (0%)",
	)

	ret := map[int32][]byte{}
	retLck := sync.Mutex{}

	f := func(role *common.TestRunRole) error {
		envID, err := t.am.PrepareAgentWithBinariesForCommit(
			role.AgentID,
			binariesInS3Path,
		)
		if err != nil {
			return err
		}
		retLck.Lock()
		ret[role.AgentID] = envID
		retLck.Unlock()
		return nil
	}
	err := t.RunForAllAgents(
		f,
		tr,
		"Deploying binaries to agents",
		time.Minute*10,
	)
	return ret, err
}
