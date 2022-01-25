package testruns

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/coordinator"
	"github.com/mit-dci/cbdc-test-controller/coordinator/agents"
	"github.com/mit-dci/cbdc-test-controller/coordinator/awsmgr"
	"github.com/mit-dci/cbdc-test-controller/coordinator/sources"
)

// Increase this if the test result calculation changed - this forces
// recalculation
// on startup of the coordinator
const TestResultVersion = 2
const PerformanceDataVersion = 3

type TestRunManager struct {
	coord                 *coordinator.Coordinator
	ev                    chan coordinator.Event
	am                    *agents.AgentsManager
	awsm                  *awsmgr.AwsManager
	src                   *sources.SourcesManager
	commitHash            string
	testRuns              []*common.TestRun // TODO: this should become persistent
	testRunsLock          sync.Mutex
	testRunResultsLock    sync.Mutex
	loadComplete          bool
	config                *TestManagerConfig
	resultCalculationChan chan resultCalculation
}

func NewTestRunManager(
	c *coordinator.Coordinator,
	am *agents.AgentsManager,
	src *sources.SourcesManager,
	ev chan coordinator.Event,
	awsm *awsmgr.AwsManager,
	commitHash string,
) (*TestRunManager, error) {
	tr := &TestRunManager{
		resultCalculationChan: make(
			chan resultCalculation,
			ParallelResultCalculation,
		),
		config:             &TestManagerConfig{MaxAgents: 1000},
		coord:              c,
		am:                 am,
		src:                src,
		ev:                 ev,
		testRunResultsLock: sync.Mutex{},
		testRuns:           []*common.TestRun{},
		testRunsLock:       sync.Mutex{},
		awsm:               awsm,
		commitHash:         commitHash,
	}
	err := tr.LoadConfig()
	if err != nil {
		return nil, err
	}

	go tr.Scheduler()

	for i := 0; i < ParallelResultCalculation; i++ {
		go tr.ResultCalculator()
	}

	return tr, nil
}

func (t *TestRunManager) RunForAllAgents(
	f func(role *common.TestRunRole) error,
	tr *common.TestRun,
	description string,
	timeout time.Duration,
) error {
	wg := sync.WaitGroup{}
	errChan := make(chan error, len(tr.Roles))
	agentsDone := int32(0)
	canceled := make(chan bool, len(tr.Roles))
	limit := 0
	for i := range tr.Roles {
		limit++
		wg.Add(1)

		go func(role *common.TestRunRole) {
			err := f(role)
			if err != nil {
				errChan <- err
			}
			select {
			case <-canceled:
				wg.Done()
				return
			default:
			}
			progress := float64(
				atomic.AddInt32(&agentsDone, 1),
			) / float64(
				len(tr.Roles),
			) * float64(
				100,
			)
			t.UpdateStatus(
				tr,
				common.TestRunStatusRunning,
				fmt.Sprintf("%s (%0.1f%%)", description, progress),
			)
			wg.Done()
		}(tr.Roles[i])
		if limit >= 200 {
			wg.Wait()
			limit = 0
		}
	}
	if !common.WaitTimeout(&wg, timeout) {
		for range tr.Roles {
			canceled <- true
		}
		return fmt.Errorf("Timed out waiting for agents")
	}
	return common.ReadErrChan(errChan)
}
