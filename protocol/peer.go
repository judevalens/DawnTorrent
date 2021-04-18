package protocol

import (
	"DawnTorrent/parser"
	"encoding/binary"
	"github.com/emirpasic/gods/maps/hashmap"
	"math"
	"net"
	"strconv"
	"sync"
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


func (peerSwarm *PeerSwarm) NewPeer(dict *parser.Dict) *Peer {
	newPeer := new(Peer)
	newPeer.id = dict.MapString["peer id"]
	newPeer.port = dict.MapString["port"]
	newPeer.ip = dict.MapString["ip"]
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
	peerDict := new(parser.Dict)
	peerDict.MapString = make(map[string]string)
	peerDict.MapString["ip"] = peerIp
	peerDict.MapString["port"] = peerPort
	peerDict.MapString["peer id"] = peerID
	newPeer := peerSwarm.NewPeer(peerDict)
	return newPeer

}