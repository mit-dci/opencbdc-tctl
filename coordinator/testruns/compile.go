package testruns

import (
	"fmt"

	"github.com/mit-dci/opencbdc-tctl/common"
)

func (t *TestRunManager) CompileBinaries(
	tr *common.TestRun,
	seeder bool,
) error {
	// Compile the binaries if needed. Needed means: the binaries for the test
	// run's requested commit hash do not exist in our binaries archive.
	seederTitle := "seeder "
	if !seeder {
		seederTitle = ""
	}
	t.UpdateStatus(
		tr,
		common.TestRunStatusRunning,
		fmt.Sprintf("Compiling %sbinaries", seederTitle),
	)
	compileProgress := make(chan float64, 1)
	done := make(chan bool, 1)
	go func() {
		for p := range compileProgress {

			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				fmt.Sprintf("Compiling %sbinaries (%.1f%%)", seederTitle, p),
			)
		}
		done <- true
	}()

	hash := tr.CommitHash
	if seeder {
		hash = tr.SeederHash
	}

	err := t.src.CompileIfNeeded(
		hash,
		(tr.RunPerf || tr.Debug) && !seeder,
		compileProgress,
	)
	<-done
	if err != nil {
		return err
	}
	return nil
}
