package protocol

import (
	"DawnTorrent/parser"
	"bytes"
	"context"
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

func (peer *Peer) stopReceiving(context context.Context){
	<- context.Done()

	err := peer.connection.Close()
	if err != nil {
		return
	}
}

func (peer *Peer) receive(context context.Context, msgChan chan BaseMsg) error {

	go peer.stopReceiving(context)

	var readFromConnError error
	msgLenBuffer := make([]byte, 4)
	for readFromConnError == nil {

			var nByteRead int
			//reads the length of the incoming msg
			nByteRead, readFromConnError = io.ReadFull(peer.connection, msgLenBuffer)
			msgLen := int(binary.BigEndian.Uint32(msgLenBuffer[0:4]))

			// reads the full payload
			incomingMsgBuffer := make([]byte, msgLen)
			nByteRead, readFromConnError = io.ReadFull(peer.connection, incomingMsgBuffer)
			_ = nByteRead
			msg, parserMsgErr := ParseMsg(bytes.Join([][]byte{msgLenBuffer, incomingMsgBuffer}, []byte{}), peer)

			if parserMsgErr == nil {
				msgChan <- msg

		}


	}

	fmt.Printf("dropping peer ..\n")

	return readFromConnError
}
