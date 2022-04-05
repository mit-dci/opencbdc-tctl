package common

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tctl/logging"
)

// TestRun describes the main type for administering and scheduling tests. The
// frontend posts a struct like this to the server to schedule a new job. Use
// the feFieldName and feFieldType decorators on the type (custom) to show the
// field in the frontend (both in the Schedule New Test Run screen and in the
// detail screen of the test runs)
type TestRun struct {
	ID                       string             `json:"id"`
	CreatedByThumbprint      string             `json:"createdByuserThumbprint"`
	Created                  time.Time          `json:"created"`
	Started                  time.Time          `json:"started"`
	Completed                time.Time          `json:"completed"`
	Status                   TestRunStatus      `json:"status"`
	CommitHash               string             `json:"commitHash"               feFieldTitle:"Code commit"                 feFieldType:"commit"`
	Architecture             string             `json:"architectureID"           feFieldTitle:"Architecture"                feFieldType:"arch"`
	BatchSize                int                `json:"batchSize"                feFieldTitle:"Batch size"                  feFieldType:"int"`
	SampleCount              int                `json:"sampleCount"              feFieldTitle:"Sample count"                feFieldType:"int"`
	ShardReplicationFactor   int                `json:"shardReplicationFactor"   feFieldTitle:"Shard Replication Factor"    feFieldType:"int"`
	STXOCacheDepth           int                `json:"stxoCacheDepth"           feFieldTitle:"STXO Cache Depth"            feFieldType:"int"`
	WindowSize               int                `json:"windowSize"               feFieldTitle:"Window size"                 feFieldType:"int"`
	TargetBlockInterval      int                `json:"targetBlockInterval"      feFieldTitle:"Target Block Interval"       feFieldType:"int"`
	ElectionTimeoutUpper     int                `json:"electionTimeoutUpper"     feFieldTitle:"Election timeout upper"      feFieldType:"int"`
	ElectionTimeoutLower     int                `json:"electionTimeoutLower"     feFieldTitle:"Election timeout lower"      feFieldType:"int"`
	Heartbeat                int                `json:"heartbeat"                feFieldTitle:"Heartbeat"                   feFieldType:"int"`
	RaftMaxBatch             int                `json:"raftMaxBatch"             feFieldTitle:"RAFT Max Batch"              feFieldType:"int"`
	SnapshotDistance         int                `json:"snapshotDistance"         feFieldTitle:"Snapshot Distance"           feFieldType:"int"`
	LoadGenOutputCount       int                `json:"loadGenOutputCount"       feFieldTitle:"Loadgen Output Count"        feFieldType:"int"`
	LoadGenInputCount        int                `json:"loadGenInputCount"        feFieldTitle:"Loadgen Input Count"         feFieldType:"int"`
	LoadGenThreads           int                `json:"loadGenThreads"           feFieldTitle:"Loadgen Threads"             feFieldType:"int"`
	BatchDelay               int                `json:"batchDelay"               feFieldTitle:"Batch Delay"                 feFieldType:"int"`
	RunPerf                  bool               `json:"runPerf"                  feFieldTitle:"Run Perf"                    feFieldType:"bool"`
	PerfSampleRate           int                `json:"perfSampleRate"           feFieldTitle:"Perf sample rate"            feFieldType:"int"`
	TrimSamplesAtStart       int                `json:"trimSamplesAtStart"       feFieldTitle:"Trim samples at start"       feFieldType:"int"`
	TrimZeroesAtStart        bool               `json:"trimZeroesAtStart"        feFieldTitle:"Trim zeroes at start"        feFieldType:"bool"`
	TrimZeroesAtEnd          bool               `json:"trimZeroesAtEnd"          feFieldTitle:"Trim zeroes at end"          feFieldType:"bool"`
	AtomizerLogLevel         string             `json:"atomizerLogLevel"         feFieldTitle:"Atomizer Log Level"          feFieldType:"loglevel"`
	ArchiverLogLevel         string             `json:"archiverLogLevel"         feFieldTitle:"Archiver Log Level"          feFieldType:"loglevel"`
	SentinelLogLevel         string             `json:"sentinelLogLevel"         feFieldTitle:"Sentinel Log Level"          feFieldType:"loglevel"`
	ShardLogLevel            string             `json:"shardLogLevel"            feFieldTitle:"Shard Log Level"             feFieldType:"loglevel"`
	CoordinatorLogLevel      string             `json:"coordinatorLogLevel"      feFieldTitle:"Coordinator Log Level"       feFieldType:"loglevel"`
	WatchtowerLogLevel       string             `json:"watchtowerLogLevel"       feFieldTitle:"Watchtower Log Level"        feFieldType:"loglevel"`
	WatchtowerBlockCacheSize int                `json:"watchtowerBlockCacheSize" feFieldTitle:"Watchtower Block Cache Size" feFieldType:"int"`
	WatchtowerErrorCacheSize int                `json:"watchtowerErrorCacheSize" feFieldTitle:"Watchtower Error Cache Size" feFieldType:"int"`
	InvalidTxRate            float64            `json:"invalidTxRate"            feFieldTitle:"Invalid TX Rate"             feFieldType:"float"`
	FixedTxRate              float64            `json:"fixedTxRate"              feFieldTitle:"Fixed TX Rate"               feFieldType:"float"`
	PreseedCount             int64              `json:"preseedCount"             feFieldTitle:"Number of preseeded outputs" feFieldType:"int"`
	PreseedShards            bool               `json:"preseedShards"            feFieldTitle:"Preseed outputs on shards"   feFieldType:"bool"`
	KeepTimedOutAgents       bool               `json:"keepTimedOutAgents"       feFieldTitle:"Keep timed out agents"       feFieldType:"bool"`
	SkipCleanUp              bool               `json:"skipCleanup"              feFieldTitle:"Skip cleanup after test"     feFieldType:"bool"`
	RetryOnFailure           bool               `json:"retryOnFailure"           feFieldTitle:"Retry on failures"           feFieldType:"bool"`
	MaxRetries               int                `json:"maxRetries"               feFieldTitle:"Maximum number of retries"   feFieldType:"int"`
	Repeat                   int                `json:"repeat"                   feFieldTitle:"Repeat test X times"         feFieldType:"int"`
	Debug                    bool               `json:"debug"                    feFieldTitle:"Run in debugger"             feFieldType:"bool"`
	RecordNetworkTraffic     bool               `json:"recordNetworkTraffic"     feFieldTitle:"Record network traffic"      feFieldType:"bool"`
	DontRunBefore            time.Time          `json:"notBefore"`
	Sweep                    string             `json:"sweep"`
	SweepID                  string             `json:"sweepID"`
	SweepRoleRuns            int                `json:"sweepRoleRuns"`
	SweepTimeMinutes         int                `json:"sweepTimeMinutes"`
	SweepTimeRuns            int                `json:"sweepTimeRuns"`
	SweepParameter           string             `json:"sweepParameterParam"`
	SweepParameterStart      float64            `json:"sweepParameterStart"`
	SweepParameterStop       float64            `json:"sweepParameterStop"`
	SweepParameterIncrement  float64            `json:"sweepParameterIncrement"`
	SweepOneAtATime          bool               `json:"sweepOneAtATime"`
	SweepRoles               []*TestRunRole     `json:"sweepRoles"`
	Priority                 int                `json:"priority"`
	Roles                    []*TestRunRole     `json:"roles"`
	Details                  string             `json:"details"`
	ExecutedCommands         []*ExecutedCommand `json:"executedCommands"`
	AgentDataAtStart         []TestRunAgentData `json:"testrunAgentData"`
	AgentDataAtEnd           []TestRunAgentData `json:"testrunAgentDataEnd"`
	PerformanceDataAvailable bool               `json:"performanceDataAvailable"`
	ControllerCommit         string             `json:"controllerCommitHash"`
	Result                   *TestResult        `json:"result"`
	SeederHash               string             `json:"seederHash"`
	TerminateChan            chan bool          `json:"-"`
	RetrySpawnChan           chan bool          `json:"-"`
	PendingResultDownloads   []S3Download       `json:"-"`
	executedCommandsLock     sync.Mutex         `json:"-"`
	DeliberateFailures       []string           `json:"-"`
	LogBuffer                string             `json:"-"`
	logLock                  sync.Mutex         `json:"-"`
	Params                   []string           `json:"-"`
	AWSInstancesStopped      bool
}

func (tr *TestRun) ReadLogTail() {
	fname := tr.LogFilePath()
	file, err := os.Open(fname)
	if err != nil {
		logging.Warnf("[Testrun %s] Unable to read logtrail: %v", tr.ID, err)
		return
	}
	defer file.Close()

	stat, err := os.Stat(fname)
	if err != nil {
		logging.Warnf("[Testrun %s] Unable to stat logfile: %v", tr.ID, err)
		return
	}
	start := stat.Size() - 4096
	if start < 0 {
		start = 0
	}
	buf := make([]byte, stat.Size()-start)
	n, err := file.ReadAt(buf, start)
	if err == nil || err == io.EOF {
		tr.LogBuffer = string(buf[:n])
	} else {
		logging.Warnf("[Testrun %s] Unable to read logtrail: %v", tr.ID, err)
	}
}
func (tr *TestRun) LogFilePath() string {
	testRunDir := filepath.Join(DataDir(), fmt.Sprintf("testruns/%s", tr.ID))
	return filepath.Join(testRunDir, "log.txt")
}
func (tr *TestRun) WriteLog(line string) {
	tr.logLock.Lock()

	// Append to (truncated) in-memory buffer
	newLog := append([]byte(tr.LogBuffer), '\n')
	newLog = append(newLog, []byte(line)...)
	if len(newLog) > 4096 {
		newLog = newLog[len(newLog)-4096:]
	}
	tr.LogBuffer = string(newLog)

	// Apppend to file on disk
	f, err := os.OpenFile(
		tr.LogFilePath(),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		logging.Warnf("[Testrun %s] Unable to append to log: %v", tr.ID, err)
	} else {
		defer f.Close()
		if _, err := f.WriteString(fmt.Sprintf("%s\n", line)); err != nil {
			logging.Warnf("[Testrun %s] Unable to append to log: %v", tr.ID, err)
		}
	}
	tr.logLock.Unlock()

	// Write to container log
	logging.Infof("[Testrun %s] %s", tr.ID, line)
}

func (tr *TestRun) LogTail() string {
	return tr.LogBuffer
}

func (tr *TestRun) FullLog() string {
	b, err := os.ReadFile(tr.LogFilePath())
	if err != nil {
		logging.Warnf("Could not read test run log: %v", err)
		return ""
	}
	return string(b)
}

type TestRunRole struct {
	Role                SystemRole          `json:"role"`
	Index               int                 `json:"roleIdx"`
	AgentID             int32               `json:"agentID"`
	AwsLaunchTemplateID string              `json:"awsLaunchTemplateID"`
	AwsAgentInstanceId  string              `json:"awsInstanceId"`
	Fail                bool                `json:"fail"`
	Failure             *TestRunRoleFailure `json:"failure"`
}

type TestRunRoleFailure struct {
	After  int  `json:"after"`
	Failed bool `json:"-"`
}

type TestRunStatus string

const TestRunStatusUnknown TestRunStatus = "Unknown"
const TestRunStatusQueued TestRunStatus = "Queued"
const TestRunStatusRunning TestRunStatus = "Running"
const TestRunStatusFailed TestRunStatus = "Failed"
const TestRunStatusCompleted TestRunStatus = "Completed"
const TestRunStatusAborted TestRunStatus = "Aborted"
const TestRunStatusInterrupted TestRunStatus = "Interrupted"
const TestRunStatusCanceled TestRunStatus = "Canceled"

type TestResultPercentile struct {
	Bucket float64 `json:"bucket"`
	Value  float64 `json:"value"`
}
type BlockLatencyResult struct {
	Average float64 `json:"avg"`
	Min     float64 `json:"min"`
	Max     float64 `json:"max"`
	StdDev  float64 `json:"std"`
}
type TestResult struct {
	ThroughputAvg         float64                       `json:"throughputAvg"`
	ThroughputStd         float64                       `json:"throughputStd"`
	ThroughputMin         float64                       `json:"throughputMin"`
	ThroughputMax         float64                       `json:"throughputMax"`
	ThroughputPercentiles []TestResultPercentile        `json:"throughputPercentiles"`
	ThroughputAvg2        float64                       `json:"throughputAvg2"`
	ThroughputAvgs        map[string]float64            `json:"throughputAvgs"`
	BlockLatencies        map[string]BlockLatencyResult `json:"blockLatencies"`

	LatencyAvg         float64                `json:"latencyAvg"`
	LatencyStd         float64                `json:"latencyStd"`
	LatencyMin         float64                `json:"latencyMin"`
	LatencyMax         float64                `json:"latencyMax"`
	LatencyPercentiles []TestResultPercentile `json:"latencyPercentiles"`
}

type MatrixResult struct {
	Config           *TestRunNormalizedConfig `json:"config"`
	Results          []*TestResult            `json:"-"`
	ResultCount      int                      `json:"resultCount"`
	ResultAvg        *TestResult              `json:"result"`
	ResultDetails    []*TestResult            `json:"resultDetails"`
	MinCompletedDate time.Time                `json:"minCompletedDate"`
	MaxCompletedDate time.Time                `json:"maxCompletedDate"`
}

type TestRunAgentData struct {
	AgentID      int32           `json:"agentID"`
	SystemInfo   AgentSystemInfo `json:"systemInfo"`
	AgentVersion string          `json:"agentVersion"`
	PingRTT      float64         `json:"pingRTT"`
	AwsRegion    string          `json:"awsRegion"`
}
