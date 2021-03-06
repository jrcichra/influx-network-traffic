package analyzer

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jrcichra/collect-network-traffic/aggregator"
	"github.com/jrcichra/collect-network-traffic/mysql"

	"github.com/jrcichra/collect-network-traffic/packet"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

//Analyzer - the main application object
type Analyzer struct {
	insertChan chan packet.Packet
	aggr       aggregator.Aggregator
}

//Start - sets up all objects for Analyzing packets and sending to mysql
func (a *Analyzer) Start(m *mysql.MySQL, interval int, interfaces ...string) {
	a.insertChan = make(chan packet.Packet)
	//Start up a packet handler for every interface
	for _, interf := range interfaces {
		log.Println("Processing traffic for interface", interf, "every", interval, "seconds")
		go a.handlePackets(interf)
	}
	//Spawn an aggregator
	a.aggr = aggregator.Aggregator{}
	a.aggr.Start(time.Duration(interval)*time.Second, a.insertChan, m)
}

//handle packets on a given interface
func (a *Analyzer) handlePackets(interf string) {
	handle, err := pcap.OpenLive(interf, 99999999, true, pcap.BlockForever)
	if err != nil {
		panic(err)
	}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	//hostname (check if we're in kubernetes)
	hostname := os.Getenv("NODE_NAME")
	if hostname == "" {
		hostname, _ = os.Hostname()
	}

	// Loop through the packet stream from the above interface
	for p := range packetSource.Packets() {
		//Make sure it's an IPV4 packet
		if ipLayer := p.Layer(layers.LayerTypeIPv4); ipLayer != nil {
			//Extract the details
			ip, _ := ipLayer.(*layers.IPv4)
			//bytes of the whole packet (not just IP layer)
			bytes := len(p.Data())
			//src ip
			src := ip.SrcIP.String()
			//dst ip
			dst := ip.DstIP.String()
			//protocol
			proto := ip.Protocol.String()
			srcPort := 0
			dstPort := 0

			// Check if it's TCP/UDP to get more data
			if tcpLayer := p.Layer(layers.LayerTypeTCP); tcpLayer != nil {
				tcp, _ := tcpLayer.(*layers.TCP)
				srcPort, _ = strconv.Atoi(tcp.SrcPort.String())
				dstPort, _ = strconv.Atoi(tcp.DstPort.String())
			} else if udpLayer := p.Layer(layers.LayerTypeUDP); udpLayer != nil {
				udp, _ := udpLayer.(*layers.UDP)
				srcPort, _ = strconv.Atoi(udp.SrcPort.String())
				dstPort, _ = strconv.Atoi(udp.DstPort.String())
			}
			pack := packet.Packet{interf, bytes, src, dst, hostname, proto, srcPort, dstPort}
			//Send this packet off for further processing
			a.insertChan <- pack
		} else {

		}

	}
}
