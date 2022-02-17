package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// environmentExists checks if the directory where the environment lives based
// on
// the ID, exists
func environmentExists(environmentID []byte) bool {
	if _, err := os.Stat(environmentDir(environmentID)); os.IsNotExist(err) {
		return false
	}
	return true
}

// environmentDir composes the directory where the environment lives based on
// the ID
func environmentDir(environmentID []byte) string {
	return filepath.Join(common.DataDir(), fmt.Sprintf("%x", environmentID))
}

// handlePrepareEnvironment handles the PrepareEnvironmentRequestMsg. Files on
// the agent are isolated using environments, which are separate folders. This
// way - even though currently not used - agents can be used to execute multiple
// test runs while not leaking file data from one run to the next by always
// starting from an empty environment (folder). The PrepareEnvironment command
// will create a new random identifier for the environment, create the
// corresponding folder and then return the ID
func (a *Agent) handlePrepareEnvironment(
	msg *wire.PrepareEnvironmentRequestMsg,
) (wire.Msg, error) {
	environmentID, err := common.RandomIDBytes(8)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(environmentDir(environmentID), 0755)
	if err != nil {
		return nil, err
	}
	return &wire.PrepareEnvironmentReplyMsg{EnvironmentID: environmentID}, nil
}

// handleDestroyEnvironment will handle the DestoryEnvironmentMsg, which results
// in deleting the full contents of the directory where the environment with the
// passed ID lives
func (a *Agent) handleDestroyEnvironment(
	msg *wire.DestroyEnvironmentMsg,
) (wire.Msg, error) {
	err := os.RemoveAll(environmentDir(msg.EnvironmentID))
	if err != nil {
		return nil, err
	}
	return &wire.AckMsg{}, nil
}

// handleRenameFile handles the RenameFileRequestMsg which is used if files are
// transferred onto the agent (for instance by downloading it from S3) that
// eventually requires to have a different name. This is used specifically with
// the shard preseeds as all members of the shard cluster download the same
// preseed file, but the name is dependent on the node ID
func (a *Agent) handleRenameFile(
	msg *wire.RenameFileRequestMsg,
) (wire.Msg, error) {
	ret := &wire.RenameFileResponseMsg{Success: true}
	sourceFile := filepath.Join(
		environmentDir(msg.EnvironmentID),
		msg.SourcePath,
	)
	targetFile := filepath.Join(
		environmentDir(msg.EnvironmentID),
		msg.TargetPath,
	)
	if _, err := os.Stat(sourceFile); os.IsNotExist(err) {
		return nil, err
	}
	err := os.Rename(sourceFile, targetFile)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
