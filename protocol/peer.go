package protocol

import (
	"DawnTorrent/parser"
	"encoding/binary"
	"net"
	"strconv"
	"time"
)



type Peer struct {
	ip                              string
	port                            string
	id                              string
	peerIndex                       int
	peerIsChocking                  bool
	peerIsInteresting               bool
	chocked                         bool
	interested                      bool
	AvailablePieces                 []bool
	numByteDownloaded               int
	time                            time.Time
	lastTimeStamp                   time.Time
	DownloadRate                    float64
	connection                      *net.TCPConn
	lastPeerPendingRequestTimeStamp time.Time
	isFree                          bool
}


func (peerSwarm *PeerSwarm) NewPeer(dict *parser.BMap) *Peer {
	newPeer := new(Peer)
	newPeer.id = dict.Strings["peer id"]
	newPeer.port = dict.Strings["port"]
	newPeer.ip = dict.Strings["ip"]
	newPeer.peerIsChocking = true
	newPeer.interested = false
	newPeer.chocked = true
	newPeer.interested = false
	newPeer.isFree = true
	return newPeer
}
func (peerSwarm *PeerSwarm) newPeerFromBytes(peer []byte) (string, string, string) {
	newPeer := new(Peer)

	ipByte := peer[0:4]

	ipString := ""
	for i := range ipByte {

		formatByte := make([]byte, 0)
		formatByte = append(formatByte, 0)
		formatByte = append(formatByte, ipByte[i])
		n := binary.BigEndian.Uint16(formatByte)
		//fmt.Printf("num byte %d\n", formatByte)
		//fmt.Printf("num %v\n", n)

		if i != len(ipByte)-1 {
			ipString += strconv.FormatUint(uint64(n), 10) + "."
		} else {
			ipString += strconv.FormatUint(uint64(n), 10)
		}
	}

	portBytes := binary.BigEndian.Uint16(peer[4:6])

	return ipString, strconv.FormatUint(uint64(portBytes), 10), ipString + ":" + newPeer.port
}
func (peerSwarm *PeerSwarm) newPeerFromStrings(peerIp, peerPort, peerID string) *Peer {
	peerDict := new(parser.BMap)
	peerDict.Strings = make(map[string]string)
	peerDict.Strings["ip"] = peerIp
	peerDict.Strings["port"] = peerPort
	peerDict.Strings["peer id"] = peerID
	newPeer := peerSwarm.NewPeer(peerDict)
	return newPeer

}