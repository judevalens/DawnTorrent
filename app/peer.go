package app

import (
	"DawnTorrent/parser"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
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

func (peer *Peer) SetConnection(conn *net.TCPConn) {
	peer.connection = conn
}

func (peer *Peer) GetId() string {
	return peer.id
}

func (peer *Peer) GetAddress() *net.TCPAddr {
	addr, err := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	if err != nil {
		return nil
	}
	return addr
}

func (peer *Peer) GetConnection() *net.TCPConn {
	return peer.connection
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
	port := strconv.FormatUint(uint64(portBytes), 10)

	log.Printf("peer addr: %v\n", ipString + ":" + port)

	return NewPeer(ipString,strconv.FormatUint(uint64(portBytes), 10),ipString + ":" + port)
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

	var err error
	msgLenBuffer := make([]byte, 4)
	for  {

			//reads the length of the incoming msg
			_, err = io.ReadFull(peer.connection, msgLenBuffer)

			msgLen := int(binary.BigEndian.Uint32(msgLenBuffer[0:4]))

			// reads the full payload
			incomingMsgBuffer := make([]byte, msgLen)
			_, err = io.ReadFull(peer.connection, incomingMsgBuffer)
			if err != nil{
				log.Fatal(err)
			}
			msg, err := ParseMsg(bytes.Join([][]byte{msgLenBuffer, incomingMsgBuffer}, []byte{}), peer)

			log.Printf("received new msg from : %v, \n %v",peer.connection.RemoteAddr().String(), msg)

			if err != nil {
			os.Exit(23)
		}

		msgChan <- msg
		

	}

	return err
}
