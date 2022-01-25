package agent

import (
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mit-dci/cbdc-test-controller/common"
	"github.com/mit-dci/cbdc-test-controller/wire"
)

// GetPublicIP uses an external site to retrieve the agent's public IP address
func GetPublicIP() (net.IP, error) {
	// TODO: This is probably not desirable
	response, err := http.Get("https://ipinfo.io/ip")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return net.ParseIP(string(responseData)), nil
}

// GetPrivateIPs reads the information of the network interfaces to return the
// private IP addresses
func GetPrivateIPs() ([]net.IP, error) {
	ips := []net.IP{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips, err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return ips, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.To4() != nil && !ip.IsLoopback() {
				ips = append(ips, ip)
			}
		}
	}
	return ips, nil
}

func GetNetworkInterfaceName() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.To4() != nil && !ip.IsLoopback() {
				return i.Name, nil
			}
		}
	}
	return "", errors.New("interface not found")
}

// GetSystemMemory reads from /proc/meminfo how much memory is available to the
// process
func GetSystemMemory() (int64, int64) {
	totalMem := int64(-1)
	availableMem := int64(-1)
	if runtime.GOOS == "linux" {
		b, err := ioutil.ReadFile("/proc/meminfo")
		if err == nil {
			output := string(b)
			for _, l := range strings.Split(output, "\n") {
				if strings.HasPrefix(l, "MemTotal:") {
					totalMem, _ = strconv.ParseInt(
						strings.TrimSpace(l[9:len(l)-3]),
						10,
						64,
					)
				} else if strings.HasPrefix(l, "MemAvailable:") {
					availableMem, _ = strconv.ParseInt(strings.TrimSpace(l[13:len(l)-3]), 10, 64)
				}
			}
		}
	}
	return totalMem, availableMem

}

// GetDiskSpace uses df -k with the DataDir as parameter to return the amount of
// diskspace available for that folder
func GetDiskSpace() int64 {
	if runtime.GOOS == "linux" {
		out, err := exec.Command("df", "-k", "--output=avail", common.DataDir()).
			Output()
		if err == nil {
			output := string(out)
			secondLine := output[strings.Index(output, "\n"):]
			kb, err := strconv.ParseInt(strings.TrimSpace(secondLine), 10, 64)
			if err == nil {
				return kb
			}
		}
	}
	return -1
}

// GetSystemInfo composes a copy of the common.AgentSystemInfo struct base on
// the current system information
func GetSystemInfo() common.AgentSystemInfo {
	hostName, err := os.Hostname()
	if err != nil {
		hostName = "unknown"
	}

	privateIPs, _ := GetPrivateIPs()
	publicIP, _ := GetPublicIP()
	totalMem, freeMem := GetSystemMemory()
	freeDisk := GetDiskSpace()

	ec2InstanceID := os.Getenv("EC2_INSTANCE_ID")

	return common.AgentSystemInfo{
		PublicIP:           publicIP,
		PrivateIPs:         privateIPs,
		AvailableDiskSpace: freeDisk,
		AvailableMemory:    freeMem,
		TotalMemory:        totalMem,
		NumCPU:             runtime.NumCPU(),
		HostName:           hostName,
		Architecture:       runtime.GOARCH,
		OperatingSystem:    runtime.GOOS,
		AWS:                len(ec2InstanceID) > 0,
		EC2InstanceID:      ec2InstanceID,
	}

}

// updateSystemInfoLoop sends a newly composed wire.UpdateSystemInfoMsg every
// five minutes to update the coordinator about the current state of this agent
func (a *Agent) updateSystemInfoLoop() {
	for {
		time.Sleep(time.Minute * 5)
		a.outgoing <- a.composeUpdateSystemInfo()
	}
}

// composeUpdateSystemInfo creates a new wire.UpdateSystemInfoMsg with the
// current system information
func (a *Agent) composeUpdateSystemInfo() *wire.UpdateSystemInfoMsg {
	return &wire.UpdateSystemInfoMsg{
		SystemInfo: GetSystemInfo(),
	}
}
