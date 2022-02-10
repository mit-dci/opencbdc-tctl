package coordinator

import (
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
)

// Event describes real-time updates sent by the coordinator to the
// frontend through a web socket. These events are used to update the
// client-side state in real time when test runs are updated
type Event struct {
	Type    EventType    `json:"type"`
	Payload EventPayload `json:"payload"`
}
type EventPayload interface{}
type EventType string

type AgentCommandRunningPayload struct {
	Command     string    `json:"command"`
	Params      []string  `json:"params"`
	Environment []string  `json:"env"`
	Started     time.Time `json:"started"`
	AgentID     int32     `json:"agentID"`
	CommandID   string    `json:"commandID"`
}

// EventTypeTestRunCreated is fired when a new test run has been created
const EventTypeTestRunCreated EventType = "testRunCreated"

type TestRunCreatedPayload struct {
	Data interface{} `json:"data"`
}

// EventTypeTestRunStatusChanged is fired when a test run's status changes
const EventTypeTestRunStatusChanged EventType = "testRunStatusChanged"

type TestRunStatusChangePayload struct {
	TestRunID string    `json:"testRunID"`
	Status    string    `json:"status"`
	Started   time.Time `json:"started"`
	Completed time.Time `json:"completed"`
	Details   string    `json:"details"`
	Debounced bool
}

// EventTypeConnectedUsersChanged is fired when a new user connects to the
// frontend or disconnects from it, and updated the number of active connected
// users
const EventTypeConnectedUsersChanged EventType = "connectedUsersChanged"

type ConnectedUsersChangedPayload struct {
	Count int `json:"count"`
}

// EventTypeMaintenanceModeChanged is fired when the system goes in or out of
// maintenance mode
const EventTypeMaintenanceModeChanged EventType = "maintenanceModeChanged"

type MaintenanceModeChangedPayload struct {
	MaintenanceMode bool `json:"maintenanceMode"`
}

// EventTypeTestRunLogAppended is fired when a test run's log has been appended
// to, but is not broadcasted to all users; only the ones that are actively
// looking at the details of the given test
const EventTypeTestRunLogAppended EventType = "testRunLogAppended"

type TestRunLogAppendedPayload struct {
	TestRunID string `json:"id"`
	Log       string `json:"log"`
}

// EventTypeTestRunResultAvailable is fired when the result calculation for a
// test run completes
const EventTypeTestRunResultAvailable EventType = "testRunResultAvailable"

type TestRunResultAvailablePayload struct {
	TestRunID string             `json:"testRunID"`
	Result    *common.TestResult `json:"result"`
}

// EventTypeSystemStateChange is fired when the system state changes. When the
// systems starts up, it is loading test runs from disk and signals this event
// when its done loading. The client is supposed to show a "System is starting
// up" notice when the system state is "loading" and only switch to the full
// UI when it's done loading
const EventTypeSystemStateChange EventType = "systemStateChange"

type SystemStateChangePayload struct {
	State string `json:"state"`
}

// EventTypeTestRunManagerConfigUpdated is fired when the coordinator's config
// has changed. Right now this config only consists of the maximum number of
// parallel agents
const EventTypeTestRunManagerConfigUpdated EventType = "testRunManagerConfigUpdated"

type TestRunManagerConfigUpdatedPayload struct {
	Config interface{} `json:"config"`
}

// EventTypeConnectedAgentCountChanged is fired when the number of connected
// agent changes by connecting or disconnecting agents
const EventTypeConnectedAgentCountChanged EventType = "agentCountChanged"

type ConnectedAgentCountChangedPayload struct {
	Count     int `json:"count"`
	Debounced bool
}

// EventTypeRedownloadComplete is fired when re-downloading the outputs from
// S3 for a particular test run has been completed
const EventTypeRedownloadComplete EventType = "redownloadComplete"

type RedownloadCompletePayload struct {
	Success   bool   `json:"success"`
	TestRunID string `json:"testRunID"`
	Error     string `json:"error"`
}
