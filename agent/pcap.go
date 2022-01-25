package agent

import (
	"io"
	"log"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/mit-dci/cbdc-test-controller/common"
)

func RecordPackets(w io.WriteCloser, stop chan bool) error {
	defer w.Close()

	// Open device
	iface, err := GetNetworkInterfaceName()
	if err != nil {
		return err
	}
	handle, err := pcap.OpenLive(iface, 256, false, 30*time.Second)
	if err != nil {
		return err
	}
	defer handle.Close()
	startTime := time.Now()
	// Use the handle as a packet source to process all packets
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		ll := packet.LinkLayer()
		nl := packet.NetworkLayer()
		tl := packet.TransportLayer()
		if ll != nil && nl != nil && tl != nil {
			if ll.LayerType() == layers.LayerTypeEthernet &&
				nl.LayerType() == layers.LayerTypeIPv4 &&
				tl.LayerType() == layers.LayerTypeTCP {

				logPacket := &common.PacketMetadata{
					SourceIP: [4]byte{0, 0, 0, 0},
					TargetIP: [4]byte{0, 0, 0, 0},
				}

				logPacket.Length = uint16(packet.Metadata().Length)
				sourceIP := net.ParseIP(nl.NetworkFlow().Src().String())
				targetIP := net.ParseIP(nl.NetworkFlow().Dst().String())

				copy(logPacket.SourceIP[:], sourceIP[12:])
				copy(logPacket.TargetIP[:], targetIP[12:])

				sourcePort, err := strconv.ParseUint(
					tl.TransportFlow().Src().String(),
					10,
					16,
				)
				if err != nil {
					log.Printf("Error parsing port: %v", err)
					continue
				}
				targetPort, err := strconv.ParseUint(
					tl.TransportFlow().Dst().String(),
					10,
					16,
				)
				if err != nil {
					log.Printf("Error parsing port: %v", err)
					continue
				}

				logPacket.SourcePort = uint16(sourcePort)
				logPacket.TargetPort = uint16(targetPort)
				logPacket.Timestamp = uint16(
					math.Round(time.Since(startTime).Seconds()),
				)
				err = logPacket.WriteTo(w)
				if err != nil {
					log.Printf("Error writing packet port: %v", err)
					continue
				}
			}
		}
		breakOut := false
		select {
		case <-stop:
			breakOut = true
		default:
		}
		if breakOut {
			break
		}
	}
	return nil
}
