package wire

import (
	"fmt"
	"time"

	"github.com/mit-dci/opencbdc-tct/common"
	"github.com/mit-dci/opencbdc-tct/logging"
)

// Receive is a utilty function to read one message from a chan of wire messages
// with a timeout of 15 seconds
func Receive(rc chan Msg) (Msg, error) {
	return ReceiveWithTimeout(rc, 15*time.Second)
}

// ReceiveWithTimeout is a utilty function to read one message from a chan of
// wire messages with a configurable timeout
func ReceiveWithTimeout(rc chan Msg, timeout time.Duration) (Msg, error) {
	var msg Msg

	logging.Debugf("Receiving from channel %v with timeout %v", rc, timeout)
	select {
	case msg = <-rc:
		logging.Debugf("Received message from channel %v", rc)
		break
	case <-time.After(timeout):
		logging.Warn(
			"Nothing received from channel %v within timeout %v",
			rc,
			timeout,
		)
		return nil, common.ErrAgentResponseTimeout
	}

	if errMsg, ok := msg.(*ErrorMsg); ok {
		return nil, fmt.Errorf("received error message: %s", errMsg.Error)
	}

	return msg, nil
}
