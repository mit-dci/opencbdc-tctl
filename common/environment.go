package common

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/mit-dci/opencbdc-tct/logging"
)

// DataDir returns the base directory where the process can store its data. This
// has been defaulted to the "data" subdirectory to the folder where the main
// process's executable is located. The directory will be created if it does not
// exist
func DataDir() string {
	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	dataDir := filepath.Join(exeDir, "data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err = os.MkdirAll(dataDir, 0755)
		if err != nil && !errors.Is(err, os.ErrExist) {
			logging.Warnf("Unable to create data directory: %v", err)
		}
	}
	return dataDir
}
