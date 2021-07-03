package core

import (
	"DawnTorrent/utils"
	"encoding/binary"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const maxPiecePerPeer = 3

type Peer struct {
	ip                              string
	port                            string
	id                              string
	peerIndex                       int
	isInteresting                   bool
	isChoked                        bool
	isChocking                      bool
	isInterested                    bool
	numByteDownloaded               int
	time                            time.Time
	lastTimeStamp                   time.Time
	DownloadRate                    float64
	connection                      *net.TCPConn
	lastPeerPendingRequestTimeStamp time.Time
	IsFree                          int64

	bitfield                        []byte
	mutex                           *sync.Mutex
	interestPoint                   int64 // how many piece I could get from him
}

func (peer *Peer) GetInterestPoint() int {
	return int(atomic.LoadInt64(&peer.interestPoint))
}

func (peer *Peer) SetInterestPoint(i int) {
	atomic.StoreInt64(&peer.interestPoint, int64(i))
}

func (peer *Peer) isAvailable(pieceIndex int) bool {
	if peer.isChocking || !peer.HasPiece(pieceIndex) || atomic.LoadInt64(&peer.IsFree) == maxPiecePerPeer {
		return false
	}
	return true
}

func (peer *Peer) UpdateBitfield(pieceIndex int) {
	byteIndex := pieceIndex / 8
	bitIndex := 7 - byteIndex%8
	if len(peer.bitfield) <= byteIndex{
		return
	}
	peer.bitfield[byteIndex] = utils.BitMask(peer.bitfield[byteIndex], 1, bitIndex)
}

func (peer *Peer) SetInterest(x bool) {
	peer.isInterested = x
}

func (peer *Peer) IsChoking() bool {
	return peer.isChocking
}

func (peer *Peer) SetChoke(x bool) {
	peer.isChocking = x
}

func (peer *Peer) IsInterested() bool {
	return peer.isInterested
}

func (peer *Peer) GetMutex() *sync.Mutex {
	return peer.mutex
}

func (peer *Peer) GetBitfield() []byte {
	return peer.bitfield
}

func (peer *Peer) SetBitField(bitfield []byte) {
	peer.bitfield = bitfield

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

func (peer *Peer) SendMsg(msg []byte) (int, error) {
	return peer.connection.Write(msg)
}

func (peer *Peer) HasPiece(pieceIndex int) bool {

	byteIndex := pieceIndex / 8

	if byteIndex >= len(peer.bitfield) {
		return false
	}

	bitIndex := 7 - (pieceIndex % 8)

	return utils.IsBitOn(peer.bitfield[byteIndex], bitIndex)

}

func NewPeer(ip, port, id string) *Peer {
	newPeer := new(Peer)
	newPeer.id = id
	newPeer.port = port
	newPeer.ip = ip
	newPeer.isChocking = true
	newPeer.isChoked = true
	newPeer.isInteresting = false
	newPeer.IsFree = 0
	newPeer.mutex = new(sync.Mutex)
	return newPeer
}
func NewPeerFromBytes(peerData []byte) *Peer {

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

	log.Printf("peer addr: %v\n", ipString+":"+port)

	return NewPeer(ipString, strconv.FormatUint(uint64(portBytes), 10), ipString+":"+port)
}