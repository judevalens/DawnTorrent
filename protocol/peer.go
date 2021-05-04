package protocol

import (
	"DawnTorrent/parser"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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

func (peer *Peer) receive(connection *net.TCPConn, peerSwarm *peerManager) error {
	var readFromConnError error
	i := 0
	msgLenBuffer := make([]byte, 4)

	for readFromConnError == nil {

		var nByteRead int
		nByteRead, readFromConnError = io.ReadFull(connection, msgLenBuffer)
		msgLen := int(binary.BigEndian.Uint32(msgLenBuffer[0:4]))
		incomingMsgBuffer := make([]byte, msgLen)

		nByteRead, readFromConnError = io.ReadFull(connection, incomingMsgBuffer)
		_ = nByteRead
		_, parserMsgErr := ParseMsg(bytes.Join([][]byte{msgLenBuffer, incomingMsgBuffer}, []byte{}), peer)

		if parserMsgErr == nil {
			/*
				// TODO will probably use a worker pool here !
				//peerSwarm.DawnTorrent.requestQueue.addJob(parsedMsg)
				if parsedMsg.ID == PieceMsg {
					//peerSwarm.DawnTorrent.pieceQueue.Add(parsedMsg)
					if nByteRead != parsedMsg.Length && nByteRead > 13 {
						fmt.Printf("nByteRead bBytesRead %v, msgLen %v", nByteRead, parsedMsg.Length)
						//os.Exit(25)
					}
					peerSwarm.torrent.msgRouter(parsedMsg)
				} else {
					peerSwarm.torrent.jobQueue.AddJob(parsedMsg)

				}
				//peerSwarm.DawnTorrent.msgRouter(parsedMsg)

				if parsedMsg.ID == UnchockeMsg {
					//os.Exit(213)
				}
			*/
		}

		i++
		//println("---------------------------------------------")

	}

	fmt.Printf("dropping peer ..\n")

	return readFromConnError
}
