package testruns

import (
	"fmt"

	"github.com/mit-dci/cbdc-test-controller/common"
)

func (t *TestRunManager) CompileBinaries(tr *common.TestRun) error {
	// Compile the binaries if needed. Needed means: the binaries for the test
	// run's requested commit hash do not exist in our binaries archive.
	t.UpdateStatus(tr, common.TestRunStatusRunning, "Compiling binaries")
	compileProgress := make(chan float64, 1)
	done := make(chan bool, 1)
	go func() {
		for p := range compileProgress {
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				fmt.Sprintf("Compiling binaries (%.1f%%)", p),
			)
		}
		done <- true
	}()
	err := t.src.CompileIfNeeded(
		tr.CommitHash,
		tr.RunPerf || tr.Debug,
		compileProgress,
	)
	<-done
	if err != nil {
		return err
	}
	return nil
}
