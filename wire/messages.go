package wire

import "github.com/mit-dci/opencbdc-tctl/common"

// HelloMsg is sent from agent to controller upon first connection. It
// identifies which version the agent is running and provides the initial system
// information of the agent
type HelloMsg struct {
	Header       MsgHeader
	SystemInfo   common.AgentSystemInfo
	AgentVersion string
}

// HelloResponseMsg is sent from controller to agent in response to HelloMsg and
// both acknowledges to the agent that the coordinator has accepted the
// HelloMsg, and tells the agent which AgentID it has on the coordinator
type HelloResponseMsg struct {
	Header      MsgHeader
	YourAgentID int32
}

// UpdateSystemInfoMsg is sent from the agent to the controller to let the
// controller know the up-to-date system information of the agent. The
// controller will respond with an AckMsg
type UpdateSystemInfoMsg struct {
	Header     MsgHeader
	SystemInfo common.AgentSystemInfo
}

// AckMsg can be sent from agent to controller or vice versa to indicate the
// message was received in scenarios where there is no information needed in the
// return path
type AckMsg struct {
	Header MsgHeader
}

// ErrorMsg can be sent from agent to controller or vice versa to indicate an
// error occurred while processing the message
type ErrorMsg struct {
	Header MsgHeader
	Error  string
}

// PrepareEnvironmentRequestMsg is sent from controller to agent to request a
// fresh environment folder on the agent in which the agent can deploy a binary
// set and execute a test run
type PrepareEnvironmentRequestMsg struct {
	Header MsgHeader
}

// PrepareEnvironmentReplyMsg is sent from agent to controller to confirm the
// succesful creation of an environment, and to inform the controller about the
// environment's unique ID under which it can be referenced in future calls that
// require an environment
type PrepareEnvironmentReplyMsg struct {
	Header        MsgHeader
	EnvironmentID []byte
}

// DestroyEnvironmentMsg is sent from controller to agent to request it to
// destroy the contents of a given environment
type DestroyEnvironmentMsg struct {
	Header        MsgHeader
	EnvironmentID []byte
}

// DeployFileFromS3RequestMsg is sent from controller to agent to ask the agent
// to download a particular object from an S3 bucket and optionally unpack it in
// the given directory within a certain environment
type DeployFileFromS3RequestMsg struct {
	Header        MsgHeader
	EnvironmentID []byte
	// The target path to download the file to, relative to the environment
	// folder
	TargetPath   string
	SourceBucket string
	SourcePath   string
	// Unpack the file if it is either a TAR or TAR.GZ
	Unpack bool
	// Ignore directory information in the archive
	FlatUnpack   bool
	SourceRegion string
	// Do not create a folder with the base name of the archive
	UnpackNoDir bool
}

// DeployFileFromS3ResponseMsg is sent from agent to controller to inform the
// controller about the success state of the deploy request
type DeployFileFromS3ResponseMsg struct {
	Header  MsgHeader
	Success bool
}

// DeployFileRequestMsg is sent from controller to agent to deploy (smaller)
// files directly over the wire protocol without first having to upload it to
// S3. This is specifically used for deploying the configuration files which are
// a few KB in size
type DeployFileRequestMsg struct {
	Header        MsgHeader
	EnvironmentID []byte
	File          common.File
	Unpack        bool
	Offset        int64
	ExpectMore    bool
	FlatUnpack    bool
}

// DeployFileResponseMsg is sent from agent to controller to inform the
// controller about the success state of the deploy request
type DeployFileResponseMsg struct {
	Header   MsgHeader
	FileHash []byte
}

// ExecuteCommandRequestMsg is sent from controller to agent to start a process
// on the agent
type ExecuteCommandRequestMsg struct {
	Header MsgHeader
	// The environment in which to execute the command
	EnvironmentID []byte
	// The directory, relative to the environment folder to use as working
	// folder
	Dir string
	// Environment variables to set (on top of the os.Environ)
	Env []string
	// The command (executable) to run
	Command string
	// The command line parameters to give to the command
	Parameters []string
	// Gather performance profiling while the command is running
	Profile bool
	// Run the command using `perf` to gather additional performance counters
	PerfProfile bool
	// The sample rate when running in perf (PerfProfile = true)
	PerfSampleRate int
	// Run in `gdb`
	Debug bool
	// The region in which the outputs bucket is
	S3OutputRegion string
	// The bucket in which to upload command outputs
	S3OutputBucket string
	// Gather bandwidth stats
	RecordNetworkTraffic bool
}

// ExecuteCommandResponseMsg is sent by the agent to the controller in response
// to a ExecuteCommandRequestMsg to inform the controller wether the command has
// been executed succesfully. Note that this does not indicate that the command
// has finished running - only that it has been started. The agent will continue
// sending ExecuteCommandStatusMsg updates, which will eventually report the
// status CommandStatusFinished when the command has finished execution.
type ExecuteCommandResponseMsg struct {
	Header MsgHeader
	// The ID under which the command is running - will also match subsequent
	// ExecyteCommandStatusMsg messages
	CommandID []byte
	// Indicates if the command was started succesfully
	Success bool
	// If Success is false, this indicates what went wrong
	Error string
}

// CommandStatus is an enumeration of the status in which commands can be
type CommandStatus int8

const (
	// The command has yielded an error
	CommandStatusError CommandStatus = -1
	// There is no status known yet
	CommandStatusUnknown CommandStatus = 0
	// The command is pending startup
	CommandStatusPending CommandStatus = 1
	// The command is running
	CommandStatusRunning CommandStatus = 2
	// The command completed running
	CommandStatusFinished CommandStatus = 3
)

// ExecuteCommandStatusMsg is sent by the agent to the controller to inform it
// of status changes
type ExecuteCommandStatusMsg struct {
	Header MsgHeader
	// The ID of the command for which an update is sent
	CommandID []byte
	// The current status of the command
	Status CommandStatus
	// If Status is CommandStatusFinished, this will contain the exit code of
	// the process
	ExitCode int
}

// PingMsg is sent from the controller to the agent, which responds with an
// AckMsg. This message is used to see if the connection to the agent is still
// alive, and how long the roundtrip of the message takes
type PingMsg struct {
	Header MsgHeader
}

// BreakCommandRequestMsg is sent from the controller to the agent to have it
// send an os.Interrupt signal to a running command identified by CommandID. The
// agent will respond with an AckMsg or ErrorMsg
type BreakCommandRequestMsg struct {
	Header MsgHeader
	// The ID of the command to send an os.Interrupt signal to
	CommandID []byte
}

// TerminateCommandRequestMsg is sent from the controller to the agent to have
// it send an os.Kill signal to a running command identified by CommandID. The
// agent will respond with an AckMsg or ErrorMsg
type TerminateCommandRequestMsg struct {
	Header MsgHeader
	// The ID of the command to send an os.Kill signal to
	CommandID []byte
}

// RenameFileRequestMsg is send from controller to agent and used to rename a
// file on the agent. The agent will respond with an RenameFileResponseMsg.
// Currently only used for renaming shard preseed files
type RenameFileRequestMsg struct {
	Header        MsgHeader
	EnvironmentID []byte
	SourcePath    string
	TargetPath    string
}

// RenameFileResponseMsg is sent from agent to controller to communicate the
// result of a Rename action
type RenameFileResponseMsg struct {
	Header  MsgHeader
	Success bool
}

// UploadFileToS3RequestMsg is sent from controller to agent to request it to
// upload a file from its file system to an S3 bucket such that the controller
// can later retrieve it. The transfer via S3 is much more efficient for many
// agents transferring files at once (as upposed to bottlenecking the built-in
// wireprotocol for this data transfer). The client sends a
// UploadFileToS3ResponseMsg in response to this request
type UploadFileToS3RequestMsg struct {
	Header        MsgHeader
	EnvironmentID []byte
	// The source path, relative to the environment's folder, to upload the file
	// from
	SourcePath   string
	TargetBucket string
	TargetPath   string
	TargetRegion string
}

// UploadFileToS3ResponseMsg is a response to UploadFileToS3RequestMsg to
// indicate the success or failure of the S3 upload
type UploadFileToS3ResponseMsg struct {
	Header  MsgHeader
	Success bool
}
