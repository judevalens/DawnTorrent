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


func  NewPeer(ip, port, id string) *Peer {
	newPeer := new(Peer)
	newPeer.id =id
	newPeer.port = port
	newPeer.ip = ip
	newPeer.peerIsChocking = true
	newPeer.interested = false
	newPeer.chocked = true
	newPeer.interested = false
	newPeer.isFree = true
	return newPeer
}
func  NewPeerFromBytes(peerData []byte) *Peer {
	newPeer := new(Peer)

	ipByte := peerData[0:4]

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

	portBytes := binary.BigEndian.Uint16(peerData[4:6])

	return NewPeer(ipString,strconv.FormatUint(uint64(portBytes), 10),ipString + ":" + newPeer.port)
}
func NewPeerFromMap(peerData *parser.BMap) *Peer{
	return NewPeer(peerData.Strings["ip"], peerData.Strings["port"],peerData.Strings["peer id"])
}