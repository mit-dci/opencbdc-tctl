package agents

import (
	"fmt"
	"os"
	"time"

	"github.com/mit-dci/opencbdc-tctl/wire"
)

// PrepareAgentWithBinariesForCommit is a convenience method that instructs
// the given agent to create a new environment, and then download the binaries
// specified by the binariesInS3 parameter into that environment and unpack it
func (am *AgentsManager) PrepareAgentWithBinariesForCommit(
	agentID int32,
	binariesInS3 string,
) ([]byte, error) {
	msg, err := am.QueryAgent(agentID, &wire.PrepareEnvironmentRequestMsg{})
	if err != nil {
		return nil, err
	}
	rep, ok := msg.(*wire.PrepareEnvironmentReplyMsg)
	if !ok {
		return nil, fmt.Errorf(
			"expected PrepareEnvironmentReplyMsg, got %T",
			rep,
		)
	}

	msg, err = am.QueryAgentWithTimeout(
		agentID,
		&wire.DeployFileFromS3RequestMsg{
			EnvironmentID: rep.EnvironmentID,
			SourceRegion:  os.Getenv("AWS_DEFAULT_REGION"),
			SourceBucket:  os.Getenv("BINARIES_S3_BUCKET"),
			SourcePath:    binariesInS3,
			TargetPath:    "sources/build.tar.gz",
			Unpack:        true,
			FlatUnpack:    false,
			UnpackNoDir:   false,
		},
		time.Minute*3,
	)
	if err != nil {
		return nil, err
	}
	rep2, ok := msg.(*wire.DeployFileFromS3ResponseMsg)
	if !ok {
		return nil, fmt.Errorf(
			"expected DeployFileFromS3ResponseMsg, got %T",
			rep2,
		)
	}

	return rep.EnvironmentID, err
}
