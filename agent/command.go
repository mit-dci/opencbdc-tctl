// Copyright (c) 2020 MIT Digital Currency Initiative,
//                    Federal Reserve Bank of Boston
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Copyright (c) 2017 David 大伟 https://github.com/struCoder/pidusage

package agent

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
	"github.com/mit-dci/opencbdc-tct/wire"
)

// handleExecuteCommand will handle the ExecuteCommandRequestMsg. This is the
// main
// logic where the agent executes the binaries that are part of the test run.
func (a *Agent) handleExecuteCommand(
	msg *wire.ExecuteCommandRequestMsg,
) (wire.Msg, error) {
	var err error

	ret := wire.ExecuteCommandResponseMsg{}
	ret.Success = true

	// Check if the environment in which we need to execute the command
	// actually exists
	if !environmentExists(msg.EnvironmentID) {
		ret.Success = false
		ret.Error = "Environment does not exist"
		return &ret, nil
	}

	// Assign a random Command ID
	ret.CommandID, err = common.RandomIDBytes(12)
	if err != nil {
		ret.Success = false
		ret.Error = "Could not create random command ID"
		return &ret, nil
	}

	logging.Debugf("Received request to execute command %s", msg.Command)

	// Replace the %ENV% placeholder in the command with the environment dir
	msg.Command = strings.ReplaceAll(
		msg.Command,
		"%ENV%",
		environmentDir(msg.EnvironmentID),
	)

	// Create a command with the passed in (%ENV% substituted) command and
	// parameters
	cmd := exec.Command(msg.Command, msg.Parameters...)
	if msg.Debug {
		// If the request asks for enabling debugging, change
		// the command to be gdb in stead, with debug.cmd being the gdb script
		// that will print proper stack traces upon exceptions that we can
		// then read using the standard error/output streams
		exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		params := []string{
			"-q", "--batch",
			"-x", filepath.Join(exeDir, "debug.cmd"),
			"--return-child-result",
			"--args",
			msg.Command,
		}
		params = append(params, msg.Parameters...)
		cmd = exec.Command("gdb", params...)
	}

	// Run the command with the environment directory as working dir
	cmd.Dir = environmentDir(msg.EnvironmentID)

	// Use the OS environment variables concatenated with the request's
	// environment
	// variables
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, msg.Env...)

	// The request message can specify another working directory relative to the
	// environment directory. If it's set, change the command's working
	// directory
	if msg.Dir != "" {
		cmd.Dir = filepath.Join(environmentDir(msg.EnvironmentID), msg.Dir)
	}

	// Open the files that we'll redirect standard out and standard error to
	outFile := filepath.Join(
		environmentDir(msg.EnvironmentID),
		fmt.Sprintf("command_%x_stdout.txt", ret.CommandID),
	)
	errFile := filepath.Join(
		environmentDir(msg.EnvironmentID),
		fmt.Sprintf("command_%x_stderr.txt", ret.CommandID),
	)

	wout, err := os.OpenFile(outFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		ret.Success = false
		ret.Error = fmt.Sprintf(
			"Failed to open stdout output file: %s",
			err.Error(),
		)
		logging.Errorf("Failed to open stdout output file: %v", err)
		return &ret, nil
	}

	werr, err := os.OpenFile(errFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		ret.Success = false
		ret.Error = fmt.Sprintf(
			"Failed to open stderr output file: %s",
			err.Error(),
		)
		logging.Errorf("Failed to open stderr output file: %v", err)
		return &ret, nil
	}

	// Set the streams to redirect stdout/stderr into the files we just created
	cmd.Stdout = wout
	cmd.Stderr = werr

	// Start the command
	err = cmd.Start()
	if err != nil {
		ret.Success = false
		ret.Error = fmt.Sprintf("Failed to start: %s", err.Error())
		logging.Errorf("Failed to start process: %v", err)
		return &ret, nil
	}

	// Create a channel to signal the command exiting to multiple subscribers
	done := make(chan bool, 5)
	go func() {
		err := cmd.Wait()
		if err != nil {
			logging.Warnf("cmd.Wait() error: %v", err)
			_, err = werr.Write(
				[]byte(fmt.Sprintf("cmd.Wait() error: %v\n", err)),
			)
			if err != nil {
				logging.Warnf(
					"Could not write cmd.Wait() error to stderr: %v",
					err,
				)
			}
		}

		time.Sleep(time.Second * 1) // allow buffers to flush

		done <- true // progress loop in this function
		done <- true // performance profiling (generic)
		done <- true // performance profiling (perf)
		done <- true // network recording
	}()
	netFile := ""
	if msg.RecordNetworkTraffic {
		netFile = filepath.Join(
			environmentDir(msg.EnvironmentID),
			fmt.Sprintf("command_%x_packets.bin", ret.CommandID),
		)
		net, err := os.OpenFile(
			netFile,
			os.O_CREATE|os.O_APPEND|os.O_WRONLY,
			0644,
		)
		if err != nil {
			ret.Success = false
			ret.Error = fmt.Sprintf(
				"Failed to open packet recording file: %s",
				err.Error(),
			)
			logging.Errorf("Failed to open packet recording file: %v", err)
			return &ret, nil
		}
		go func() {
			err := RecordPackets(net, done)
			if err != nil {
				logging.Errorf("Error recording packets: %v", err)
			}
		}()
	}

	processID := cmd.Process.Pid
	if msg.Profile {
		// If we want to profile the performance while the command is running,
		// then execute this in a separate goroutine. Pass in the done channel
		// to signal to this goroutine that the command has exited.
		go a.profilePerformance(
			msg.EnvironmentID,
			ret.CommandID,
			processID,
			done,
		)
	}

	perfWg := sync.WaitGroup{}
	if msg.PerfProfile {
		// If we want to profile the performance using perf traces, then
		// execute this in a separate goroutine. Pass in the done channel
		// to signal to this goroutine that the command has exited. Pass in
		// the waitgroup to be able to wait for perf to finish its business
		// before uploading everything to the S3 buckets. Once the command is
		// completed, we kill perf but it'll still need some time to complete
		// building the traces - which we should wait for.
		go a.profilePerformancePerf(
			msg.EnvironmentID,
			ret.CommandID,
			processID,
			done,
			&perfWg,
			msg.PerfSampleRate,
		)
	}

	// Send an initial command status to the coordinator to show that the
	// command is now running
	a.outgoing <- &wire.ExecuteCommandStatusMsg{
		CommandID: ret.CommandID,
		Status:    wire.CommandStatusRunning,
	}

	// Insert the pending command into our pendingCommands array
	a.addPendingCommand(&pendingCommand{cmd: cmd, id: ret.CommandID})

	// Monitor the completion of the process in a separate goroutine - the main
	// process loop should return the result to the ExecuteCommand request to
	// signal the message has been properly handled and the process is running.
	go func() {
		for {
			exit := false
			// Monitor the done signal or a 20 second timeout
			select {
			case <-done:
				exit = true
			case <-time.After(20 * time.Second):
			}
			if exit {
				break
			}
			// If the process isn't done, this code is reached every 20 seconds
			// and signals to the controller that we're still alive and the
			// process we spawned is still running.
			a.outgoing <- &wire.ExecuteCommandStatusMsg{
				CommandID: ret.CommandID,
				Status:    wire.CommandStatusRunning,
			}
		}

		// The process has now completed. First, wait for the perf profile
		// to complete processing (if relevant) - this Wait() call will complete
		// immediately if perf profiling is not enabled.
		perfWg.Wait()

		// Close the stderr and stdout file streams
		werr.Close()
		wout.Close()

		// Upload the command's stdout and stderr to S3

		err := a.uploadFileToS3(
			outFile,
			msg.S3OutputRegion,
			msg.S3OutputBucket,
			fmt.Sprintf(
				"command-outputs/%x/cmd_%x_stdout.txt",
				ret.CommandID[:4],
				ret.CommandID,
			),
		)
		if err != nil {
			logging.Warnf("Could not upload command stdout to S3: %v", err)
		}

		err = a.uploadFileToS3(
			errFile,
			msg.S3OutputRegion,
			msg.S3OutputBucket,
			fmt.Sprintf(
				"command-outputs/%x/cmd_%x_stderr.txt",
				ret.CommandID[:4],
				ret.CommandID,
			),
		)
		if err != nil {
			logging.Warnf("Could not upload command stderr to S3: %v", err)
		}

		if msg.RecordNetworkTraffic {
			err = a.uploadFileToS3(
				netFile,
				msg.S3OutputRegion,
				msg.S3OutputBucket,
				fmt.Sprintf(
					"command-outputs/%x/cmd_%x_packets.bin",
					ret.CommandID[:4],
					ret.CommandID,
				),
			)
			if err != nil {
				logging.Warnf(
					"Could not upload command network packets to S3: %v",
					err,
				)
			}
		}

		// Report to the controller that the command has completed
		a.outgoing <- &wire.ExecuteCommandStatusMsg{
			CommandID: ret.CommandID,
			Status:    wire.CommandStatusFinished,
			ExitCode:  cmd.ProcessState.ExitCode(),
		}

		// Remove the command from the pendingCommands array
		a.deletePendingCommand(ret.CommandID)
	}()

	return &ret, nil
}

// This functions runs a `perf` performance profile on the given process
func (a *Agent) profilePerformancePerf(
	environmentID, commandID []byte,
	processID int,
	done chan bool,
	wg *sync.WaitGroup,
	sampleRate int,
) {
	// See the man page for perf for more info:
	// https://man7.org/linux/man-pages/man1/perf.1.html
	// This creates a command to execute perf record with the process ID of the
	// process that was just kicked off, with the requested sample rate
	cmd := exec.Command(
		"perf",
		"record", // The command to run (record)
		"-F",     // -F <Samples per second> defines the sample rate
		fmt.Sprintf("%d", sampleRate),
		"-p", // -p <PID> defines the process to record
		fmt.Sprintf("%d", processID),
		"-g", // Capture call graphs (stack traces)
		"-o", // Output file
		fmt.Sprintf("perf_%x.data", commandID),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Dir = environmentDir(environmentID)
	logging.Infof("Starting perf")

	err := cmd.Start()
	if err != nil {
		logging.Errorf("Could not start perf: %v", err)
		return
	}

	logging.Infof("Started perf")
	// Signal that we're waiting on perf to complete to the waitgroup.
	// this causes the main logic to wait for the processing to complete
	// which can take a while after the main process is done.
	wg.Add(1)
	// Once the entire function completed, signal its completion to the
	// waitgroup
	defer wg.Done()

	// Wait for the done channel to signal that the main
	// process has completed. At that time, send an interrupt
	// signal to the perf process to trigger its completion
	// do this in a separate routine, such that the main routine
	// can wait for the process to complete after this interrupt has
	// been sent.
	go func() {
		for {
			exit := false
			select {
			case <-done:
				exit = true
			case <-time.After(time.Second):
			default:
			}
			if exit {
				logging.Infof("Interrupting perf")
				if cmd.Process != nil {
					err = cmd.Process.Signal(os.Interrupt)
					if err != nil {
						logging.Errorf("Could not send interrupt: %v", err)
					}
				}
				break
			}
		}
	}()

	// Wait for the perf command to complete. Completion is triggered by sending
	// the interrupt signal upon the main process completing.
	err = cmd.Wait()
	if err != nil {
		logging.Errorf("Perf cmd.Wait() error: %v", err)
	}

	// Execute the perf-archive script to gather all the necessary data to
	// generate the perf script below. Perf script results in the necessary data
	// to produce flame graphs
	exeDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	cmd2 := exec.Command(
		"bash",
		filepath.Join(exeDir, "perf-archive.sh"),
		fmt.Sprintf("perf_%x.data", commandID),
	)
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stdout
	cmd2.Dir = environmentDir(environmentID)
	logging.Infof("Starting perf archive")
	err = cmd2.Start()
	if err != nil {
		logging.Errorf("Could not run perf archive: %v", err)
		return
	}

	err = cmd2.Wait()
	if err != nil {
		logging.Errorf("Perf archive cmd.Wait() error: %v", err)
		return
	}

	// Execute perf script to archive the data nessary to produce flame graphs.
	// When executing the generation of flame graphs on a machine that is not
	// the same as where perf was run, you need to carry certain information
	// which is contained in this file.
	script, err := os.OpenFile(
		filepath.Join(
			environmentDir(environmentID),
			fmt.Sprintf("perf_%x.script", commandID),
		),
		os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		logging.Errorf("Error opening script output file: %v", err)
		return
	}
	defer script.Close()
	cmd3 := exec.Command(
		"perf",
		"script",
		"-i",
		fmt.Sprintf("perf_%x.data", commandID),
	)
	cmd3.Stdout = script
	cmd3.Stderr = os.Stdout
	cmd3.Dir = environmentDir(environmentID)
	err = cmd3.Start()
	if err != nil {
		logging.Errorf("Could not run perf script: %v", err)
		return
	}

	err = cmd3.Wait()
	if err != nil {
		logging.Errorf("Perf script cmd.Wait() error: %v", err)
		return
	}
	logging.Infof("Finished perf, perf archive and perf script")
}

func FormatStdOut(stdout []byte, userfulIndex int) []string {
	infoArr := strings.Split(string(stdout), "\n")[userfulIndex]
	ret := strings.Fields(infoArr)
	return ret
}

// profilePerformance is the method to gather system performance metrics on the
// agent every second while the main process is running
func (a *Agent) profilePerformance(
	environmentID, commandID []byte,
	processID int,
	done chan bool,
) {
	// Open a plain text file to write the performance data to
	perf, err := os.OpenFile(
		filepath.Join(
			environmentDir(environmentID),
			fmt.Sprintf("perf_%x.txt", commandID),
		),
		os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		logging.Errorf("Could not write to performance output file")
		return
	}
	defer perf.Close()

	// clkTck and pageSize are system parameters that determine the values for
	// cpu usage and memory usage
	var clkTck float64 = 100
	var pageSize float64 = 4096

	// Read clkTck using the getconf linux command. If it fails we use the
	// default value above.
	clkTckStdout, err := exec.Command("getconf", "CLK_TCK").Output()
	if err == nil {
		clkTck, _ = strconv.ParseFloat(FormatStdOut(clkTckStdout, 0)[0], 64)
	}

	// Read pageSize using the getconf linux command. If it fails we use the
	// default value above
	pageSizeStdout, err := exec.Command("getconf", "PAGESIZE").Output()
	if err == nil {
		pageSize, _ = strconv.ParseFloat(FormatStdOut(pageSizeStdout, 0)[0], 64)
	}

	// Write the clkTck and pageSize variables to the output file once - they
	// don't change during the course of the test run
	if _, err := perf.Write([]byte(fmt.Sprintf("%%CLK_TCK %f\n", clkTck))); err != nil {
		logging.Warnf("Error writing perf: %v", err)
	}
	if _, err := perf.Write([]byte(fmt.Sprintf("%%PAGESIZE %f\n", pageSize))); err != nil {
		logging.Warnf("Error writing perf: %v", err)
	}

	for {
		// If the main process is complete, signaled through the done channel,
		// we can stop gathering performance metrics.
		select {
		case <-done:
			return
		case <-time.After(1 * time.Second):
		}

		// Convenience method to write raw bytes to the performance trace file
		includeData := func(title string, b []byte) {
			if _, err := perf.Write([]byte(fmt.Sprintf("\n%%%s\n", title))); err != nil {
				logging.Warnf("Error writing perf: %v", err)
			}
			if _, err := perf.Write(b); err != nil {
				logging.Warnf("Error writing perf: %v", err)
			}
			if _, err := perf.Write([]byte(fmt.Sprintf("\n%%END-%s\n", title))); err != nil {
				logging.Warnf("Error writing perf: %v", err)
			}
		}

		// Convenience method to snapshot the contents of a file on the system
		// into the performance trace file
		includeDataFile := func(title, filePath string) {
			b, _ := ioutil.ReadFile(filePath)
			includeData(title, b)
		}

		// Convenience method to run a command and write its output into the
		// performance trace file
		includeDataCommand := func(title, name string, args ...string) {
			b, _ := exec.Command(name, args...).Output()
			includeData(title, b)
		}

		// Signal the start of the sample with a timestamp
		if _, err := perf.Write(
			[]byte(fmt.Sprintf("\n%%SAMPLE_START %d\n", time.Now().UnixNano())),
		); err != nil {
			logging.Warnf("Error writing perf: %v", err)
		}

		// Include the uptime
		includeDataFile("UPTIME", path.Join("/proc", "uptime"))
		// Include the CPU usage overall
		includeDataFile("STAT", path.Join("/proc", "stat"))
		// Include the CPU usage of the monitored process
		includeDataFile(
			"STAT-CHILD",
			path.Join("/proc", fmt.Sprintf("%d", processID), "stat"),
		)
		// Include the memory usage of the monitored process
		includeDataFile(
			"STATM-CHILD",
			path.Join("/proc", fmt.Sprintf("%d", processID), "statm"),
		)
		// Include the CPU usage of the agent process
		includeDataFile(
			"STAT-AGENT",
			path.Join("/proc", fmt.Sprintf("%d", os.Getpid()), "stat"),
		)
		// Include the memory usage of the agent process
		includeDataFile(
			"STATM-AGENT",
			path.Join("/proc", fmt.Sprintf("%d", os.Getpid()), "statm"),
		)

		// Include the network buffer sizes
		b, _ := a.getNetBuffers(processID)
		includeData("NETBUF-CHILD", b)

		// Include the overall system memory status
		includeDataCommand("FREE", "free")

		// Include the free disk space in the environment directory
		includeDataCommand(
			"DISKENV",
			"df",
			"-kT",
			environmentDir(environmentID),
		)

		// Include the free disk space on the entire machine
		includeDataCommand("DISKALL", "df", "-k")

		// Write end sample marker with timestamp
		if _, err := perf.Write(
			[]byte(fmt.Sprintf("\n%%SAMPLE_END %d\n", time.Now().UnixNano())),
		); err != nil {
			logging.Warnf("Error writing perf: %v", err)
		}
	}
}

// getNetBuffers reads the current network buffers on the machine using the `ss`
// command
func (a *Agent) getNetBuffers(pid int) ([]byte, error) {
	ssout, err := exec.Command("ss", "--tcp", "-p", "-n").Output()
	if err != nil {
		return []byte{}, err
	}

	var buf bytes.Buffer

	lines := strings.Split(string(ssout), "\n")
	for _, l := range lines[1:] { // Skip header
		if strings.Contains(l, fmt.Sprintf("pid=%d,", pid)) {
			fields := strings.Fields(l)
			buf.Write(
				[]byte(
					fmt.Sprintf(
						"%s\t%s\t%s\t%s\t%s\n",
						fields[0],
						fields[1],
						fields[2],
						fields[3],
						fields[4],
					),
				),
			)
		}
	}
	return buf.Bytes(), nil
}

// handleBreakCommand handles the BreakCommandRequestMsg. This message is used
// to instruct the agent to send an interrupt signal to the command idenfified
// by its ID
func (a *Agent) handleBreakCommand(
	msg *wire.BreakCommandRequestMsg,
) (wire.Msg, error) {
	cmd, ok := a.getPendingExecutingCommand(msg.CommandID)
	if !ok {
		logging.Warnf(
			"Coordinator asked to break command %x which is unknown",
			msg.CommandID,
		)
	}
	if ok && cmd == nil {
		logging.Warnf(
			"Coordinator asked to break command %x which is nil",
			msg.CommandID,
		)
	}
	if ok && cmd != nil && cmd.Process != nil {
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			logging.Warnf("Error sending Interrupt signal: %v", err)
		}
	}
	return &wire.AckMsg{}, nil
}

// handleTerminateCommand handles the TerminateCommandRequestMsg. This message
// is used
// to instruct the agent to send a kill signal to the command idenfified
// by its ID
func (a *Agent) handleTerminateCommand(
	msg *wire.TerminateCommandRequestMsg,
) (wire.Msg, error) {
	cmd, ok := a.getPendingExecutingCommand(msg.CommandID)
	if !ok {
		logging.Warnf(
			"Coordinator asked to terminate command %x which is unknown",
			msg.CommandID,
		)
	}
	if ok && cmd == nil {
		logging.Warnf(
			"Coordinator asked to terminate command %x which is nil",
			msg.CommandID,
		)
	}
	if ok && cmd != nil && cmd.Process != nil {
		err := cmd.Process.Signal(os.Kill)
		if err != nil {
			logging.Warnf("Error sending Kill signal: %v", err)
		}
	}
	return &wire.AckMsg{}, nil
}

// addPendingCommand acquires a lock on the pendingCommands array and inserts
// a new command into it
func (a *Agent) addPendingCommand(cmd *pendingCommand) {
	a.pendingCommandsLock.Lock()
	defer a.pendingCommandsLock.Unlock()
	a.pendingCommands = append(a.pendingCommands, cmd)
}

// addPendingCommand acquires a lock on the pendingCommands array and removes
// the command identified by the passed id from it
func (a *Agent) deletePendingCommand(id []byte) {
	a.pendingCommandsLock.Lock()
	defer a.pendingCommandsLock.Unlock()
	newPendingCommands := []*pendingCommand{}
	for _, c := range a.pendingCommands {
		if !bytes.Equal(c.id, id) {
			newPendingCommands = append(newPendingCommands, c)
		}
	}
	a.pendingCommands = newPendingCommands
}

// getPendingCommand returns the command identified by the given ID
func (a *Agent) getPendingExecutingCommand(id []byte) (*exec.Cmd, bool) {
	for _, c := range a.pendingCommands {
		if bytes.Equal(c.id, id) {
			return c.cmd, true
		}
	}
	return nil, false
}
