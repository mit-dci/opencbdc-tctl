package agent

import (
	"os/exec"
	"syscall"

	"github.com/mit-dci/opencbdc-tctl/logging"
)

const DesiredULimit = 1 * 1024 * 1024

func IncreaseULimit() {
	var limit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &limit)
	if err != nil {
		logging.Errorf("Error Getting ulimit ", err)
	}
	logging.Infof("Current ulimit: %v", limit)

	// First increase current to max - don't need root for that
	if limit.Max > limit.Cur {
		limit.Cur = limit.Max
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit)
		if err != nil {
			logging.Errorf("Error Setting ulimit ", err)
			return
		}
		logging.Infof("Changed ulimit to: %v", limit)
	}

	// Now try even further increase if needed
	if limit.Max < DesiredULimit {
		limit.Max = DesiredULimit
		limit.Cur = limit.Max
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit)
		if err != nil {
			logging.Errorf("Error Setting ulimit higher ", err)
			return
		}
		logging.Infof("Changed ulimit to: %v", limit)
	}
}

func CheckUlimit() {
	cmd := exec.Command(
		"bash",
		"-c",
		"ulimit -n",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logging.Errorf("Error checking ulimit in spawned process: %v", err)
	}
	logging.Infof("Output of ulimit -n: %s", string(out))
}
