package common

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type AgentSystemInfo struct {
	HostName           string   `json:"hostname"`
	PublicIP           net.IP   `json:"publicIP"`
	PrivateIPs         []net.IP `json:"privateIPs"`
	AvailableDiskSpace int64    `json:"diskAvailable"`
	TotalMemory        int64    `json:"memTotal"`
	AvailableMemory    int64    `json:"memAvailable"`
	OperatingSystem    string   `json:"os"`
	Architecture       string   `json:"arch"`
	NumCPU             int      `json:"numCPU"`
	AWS                bool     `json:"aws"`
	EC2InstanceID      string   `json:"ec2InstanceId"`
}

type File struct {
	FilePath string
	Contents []byte
}

func (a AgentSystemInfo) ToString() string {
	var buf bytes.Buffer
	format := "%30s : %v\n"
	fmt.Fprintf(&buf, format, "Public IP", a.PublicIP)
	fmt.Fprintf(&buf, format, "Private IPs", a.PrivateIPs)
	fmt.Fprintf(&buf, format, "Available disk space (kB)", a.AvailableDiskSpace)
	fmt.Fprintf(&buf, format, "Total system memory (kB)", a.TotalMemory)
	fmt.Fprintf(&buf, format, "Available system memory (kB)", a.AvailableMemory)
	fmt.Fprintf(&buf, format, "Operating System", a.OperatingSystem)
	fmt.Fprintf(&buf, format, "Architecture", a.Architecture)
	fmt.Fprintf(&buf, format, "Number of CPUs", a.NumCPU)
	if a.AWS {
		fmt.Fprintf(&buf, format, "Running in AWS", "Yes")
		fmt.Fprintf(&buf, format, "EC2 Instance ID", a.EC2InstanceID)
	} else {
		fmt.Fprintf(&buf, format, "Running in AWS", "No")
	}
	return buf.String()
}

type S3Download struct {
	SourceRegion string
	SourceBucket string
	SourcePath   string
	TargetPath   string
	Retries      int
}

type S3Upload struct {
	TargetRegion string
	TargetBucket string
	TargetPath   string
	SourcePath   string
	Retries      int
}

type PacketMetadata struct {
	Length     uint16
	SourceIP   [4]byte
	TargetIP   [4]byte
	SourcePort uint16
	TargetPort uint16
	Timestamp  uint16
}

func (p *PacketMetadata) WriteTo(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, p.Length)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, p.SourceIP)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, p.SourcePort)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, p.TargetIP)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, p.TargetPort)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, p.Timestamp)
	if err != nil {
		return err
	}
	return nil
}

func ReadPacketMetadata(r io.Reader) (*PacketMetadata, error) {
	p := &PacketMetadata{}
	err := binary.Read(r, binary.BigEndian, &p.Length)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &p.SourceIP)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &p.SourcePort)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &p.TargetIP)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &p.TargetPort)
	if err != nil {
		return nil, err
	}

	err = binary.Read(r, binary.BigEndian, &p.Timestamp)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *PacketMetadata) String() string {
	return fmt.Sprintf(
		"Len=%d Src=%s:%d Dst=%s:%d Time=%d",
		p.Length,
		net.IP(p.SourceIP[:]).String(),
		p.SourcePort,
		net.IP(p.TargetIP[:]).String(),
		p.TargetPort,
		p.Timestamp,
	)
}
