package testruns

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/mit-dci/opencbdc-tctl/common"
)

// PortIncrement is a type that specifies the offset from the default port
// based on predefined purposes
type PortIncrement int

// PortIncrementDefaultPort is the offset from the standard port for default
// listeners. These are the listeners for the system role's default work, which
// in most cases (Atomizer, Shard, Sentinel) is receiving transactions for
// processing.
const PortIncrementDefaultPort PortIncrement = 0

// PortIncrementRaftPort is the offset from the standard port for the RAFT
// listeners. This is used by RAFT replicated components such as the Atomizer
// and 2PC Shards/Coordinators to communicate between cluster participants
const PortIncrementRaftPort PortIncrement = 1

// PortIncrementClientPort is the offset from the standard port for the Client
// listeners (for instance the endpoints the shards expose allowing clients to
// query the spent status of a UHS)
const PortIncrementClientPort PortIncrement = 2

// portNums dictates the port numbers the controller will use for each of these
// system components. These port numbers are specified in the configuration file
// which will both be used by the component itself to activate the proper port
// to listen for incoming connections, as well as the other roles to connect to
// peers. Since we are not running two system roles on the same machine these
// ports can safely overlap
var portNums = map[common.SystemRole]int{
	common.SystemRoleRaftAtomizer:     5001,
	common.SystemRoleCoordinator:      5001,
	common.SystemRoleShard:            5002,
	common.SystemRoleShardTwoPhase:    5002,
	common.SystemRoleSentinel:         5003,
	common.SystemRoleSentinelTwoPhase: 5003,
	common.SystemRoleArchiver:         5004,
	common.SystemRoleWatchtower:       5005,
}

// GetRoleEndpoint will return the IP and port at which a particular role in our
// test is / should be listening. This endpoint is derived from the IP address
// reported by the agent and the port number based on the default for that role,
// and the specified increment (Default, RAFT or Client)
func (t *TestRunManager) GetRoleEndpoint(
	tr *common.TestRun,
	role *common.TestRunRole,
	portIncrement PortIncrement,
) (string, error) {
	// Get the agent at which this role will run
	a, err := t.coord.GetAgent(role.AgentID)
	if err != nil {
		return "", err
	}
	// Calculate the port number from the base in the portNums map and the
	// increment specified
	portnum := portNums[role.Role] + int(portIncrement)

	// Return the endpoint based on the agent's IP information and the
	// calculated port number
	return fmt.Sprintf("%s:%d", a.SystemInfo.PrivateIPs[0], portnum), nil
}

// WaitForRolesOnline will
func (t *TestRunManager) WaitForRolesOnline(
	tr *common.TestRun,
	roles []*common.TestRunRole,
	portIncrement PortIncrement,
	timeout time.Duration,
) error {

	wg := sync.WaitGroup{}
	errs := make([]error, 0)
	errLock := sync.Mutex{}

	for i := range roles {
		wg.Add(1)
		go func(rl *common.TestRunRole) {
			err := t.WaitForRoleOnline(tr, rl, portIncrement, timeout)
			if err != nil {
				errLock.Lock()
				errs = append(errs, err)
				errLock.Unlock()
			}
			wg.Done()
		}(roles[i])
	}

	wg.Wait()
	if len(errs) > 0 {
		jointErr := ""
		for _, e := range errs {
			jointErr += e.Error() + "\n"
		}
		return errors.New("Failed waiting for roles: " + jointErr)
	}
	return nil
}

// WaitForRoleOnline will wait until it's able to open a TCP connection to the
// endpoint of the role at the given `portIncrement`. This can for instance be
// used to wait for the RAFT port of a follower node to be online before
// continuing the start sequence and start the leader of that RAFT cluster. It
// does not do any semantic check if the TCP endpoint is actually processing
// data. It just opens and closes the connection - once it can succesfully open
// the connection the method returns. If the `timeout` specified has elapsed and
// no connection is possible, the method returns an error
func (t *TestRunManager) WaitForRoleOnline(
	tr *common.TestRun,
	role *common.TestRunRole,
	portIncrement PortIncrement,
	timeout time.Duration,
) error {
	// Use GetRoleEndpoint to determine the IP and port we need to connect to.
	endpoint, err := t.GetRoleEndpoint(tr, role, portIncrement)
	if err != nil {
		return err
	}

	// Try connecting to the endpoint until success or timeout
	start := time.Now()
	dialTimeout := timeout / 4
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf(
				"timeout waiting for %s %d to be online on endpoint %s",
				role.Role,
				role.Index,
				endpoint,
			)
		}
		_, err := net.DialTimeout("tcp", endpoint, dialTimeout)
		if err == nil {
			break
		}
	}
	return nil
}
