package common

import (
	"errors"
	"fmt"
)

var ErrWrongMessageType = errors.New("received wrong message type")

var ErrAgentResponseTimeout = errors.New(
	"agent did not respond within timeout limit",
)
var ErrCommandIDNotFound = errors.New("command ID not found")
var ErrRunNotFound = errors.New("run not found")

// ReadErrChan reads a channel of errors until it's closed and wraps it
// in a single error (or nil if no errors were sent to the channel)
func ReadErrChan(c chan error) error {
	close(c)
	errStr := ""
	errs := 0
	for err := range c {
		errStr += err.Error() + "\n"
		errs++
	}
	if errs == 0 {
		return nil
	}
	return fmt.Errorf("%d errors occurred:\r\n%s", errs, errStr)
}
