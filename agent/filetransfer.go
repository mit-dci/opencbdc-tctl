package agent

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tctl/common"
	"github.com/mit-dci/opencbdc-tctl/logging"
	"github.com/mit-dci/opencbdc-tctl/wire"
)

// handleDeployFile handles the DeployFileRequestMsg which contains (part of)
// the file's contents and the path / offset at which to write it.
func (a *Agent) handleDeployFile(
	msg *wire.DeployFileRequestMsg,
) (wire.Msg, error) {
	ret := &wire.DeployFileResponseMsg{}
	logging.Debugf(
		"Received %d bytes file to deploy at %s offset %d - Expect more: %t",
		len(msg.File.Contents),
		msg.File.FilePath,
		msg.Offset,
		msg.ExpectMore,
	)

	// Compose the full target path from the environment directory and the
	// path specified in the request message
	targetFile := filepath.Join(
		environmentDir(msg.EnvironmentID),
		msg.File.FilePath,
	)

	// Ensure the directory where the file must go exists
	err := os.MkdirAll(filepath.Dir(targetFile), 0755)
	if err != nil {
		return nil, err
	}

	// Open the target file for writing
	f, err := os.OpenFile(targetFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// If this is an incremental write (chunked content), seek to the offset
	// position at which to commence writing
	_, err = f.Seek(msg.Offset, 0)
	if err != nil {
		return nil, err
	}

	// Write the content and check all bytes have been written
	n, err := f.Write(msg.File.Contents)
	if err != nil {
		return nil, err
	}
	if n != len(msg.File.Contents) {
		return nil, fmt.Errorf(
			"Expected to write %d bytes, but did %d in stead",
			len(msg.File.Contents),
			n,
		)
	}

	// Unpack the file if this is the final chunk and unpacking has been
	// requested
	if msg.Unpack && !msg.ExpectMore {
		err = common.TarExtractFlat(targetFile, msg.FlatUnpack, false)
		if err != nil {
			return nil, fmt.Errorf(
				"Error extracting file %s: %v",
				msg.File.FilePath,
				err,
			)
		}
		os.Remove(targetFile)
	}

	return ret, nil
}
