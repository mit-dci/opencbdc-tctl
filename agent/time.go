package agent

import (
	"os/exec"
	"time"

	"github.com/beevik/ntp"
	"github.com/mit-dci/opencbdc-tct/logging"
)

// SyncTime fetches the latest time from an NTP server and then tries to set the
// system's clock to it
func (a *Agent) SyncTime() {
	logging.Info("Syncing date/time")
	ntpTime, err := ntp.Time("169.254.169.123")
	if err != nil {
		logging.Errorf("Error getting time from NTP: %v", err)
	} else {
		logging.Infof("Got time from NTP: %v - System time: %v", ntpTime, time.Now())
		err = SetSystemTime(ntpTime)
		if err != nil {
			logging.Errorf("Error setting system date: %v", err)
		}
	}
}

// SetSystemTime uses the system's date binary to set the current system time
// to the given newTime
func SetSystemTime(newTime time.Time) error {
	_, lookErr := exec.LookPath("date")
	if lookErr != nil {
		logging.Errorf(
			"Date binary not found, cannot set system date: %s\n",
			lookErr.Error(),
		)
		return lookErr
	} else {
		//dateString := newTime.Format("2006-01-2 15:4:5")
		dateString := newTime.Format("2 Jan 2006 15:04:05")
		logging.Infof("Setting system date to: %s\n", dateString)
		args := []string{"--set", dateString}
		return exec.Command("date", args...).Run()
	}
}
