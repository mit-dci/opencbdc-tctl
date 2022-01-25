package common

import "time"

type ExecutedCommand struct {
	AgentID       int32     `json:"agentID"`
	CommandID     string    `json:"commandID"`
	ExitCode      int       `json:"returnCode"`
	Description   string    `json:"description"`
	InternalError string    `json:"internalError"`
	Params        []string  `json:"params"`
	Environment   []string  `json:"env"`
	Completed     time.Time `json:"completed"`
	Stdout        string    `json:"-"` // don't serialize this by default - fetch through separate API
	Stderr        string    `json:"-"` // don't serialize this by default - fetch through separate API
}

func (tr *TestRun) AddExecutedCommand(cmd *ExecutedCommand) {
	tr.executedCommandsLock.Lock()
	cmd.Completed = time.Now()
	tr.ExecutedCommands = append(tr.ExecutedCommands, cmd)
	tr.executedCommandsLock.Unlock()
}
