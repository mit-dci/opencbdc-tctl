package testruns

import (
	"fmt"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/coordinator"
)

// WriteLog writes a statement to a testrun's log file and sends it over the
// real-time channel to the UI
func (t *TestRunManager) WriteLog(
	tr *common.TestRun,
	format string,
	a ...interface{},
) {
	line := fmt.Sprintf(
		"[%s] %s",
		time.Now().Format("2006-01-02 15:04:05.999"),
		fmt.Sprintf(format, a...),
	)
	t.ev <- coordinator.Event{
		Type: coordinator.EventTypeTestRunLogAppended,
		Payload: coordinator.TestRunLogAppendedPayload{
			TestRunID: tr.ID,
			Log:       line + "\n",
		},
	}
	tr.WriteLog(line)
}
