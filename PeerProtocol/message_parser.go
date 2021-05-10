package PeerProtocol

import (
	"DawnTorrent/JobQueue"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	//	"strconv"
	//"encoding/binary"
)



const (
	HandShakeMsgID            = 19
	ChockedMsg                = 0
	UnchockeMsg               = 1
	InterestedMsg             = 2
	UninterestedMsg           = 3
	HaveMsg                   = 4
	BitfieldMsg               = 5
	RequestMsg                = 6
	PieceMsg                  = 7
	CancelMsg                 = 8
	DefaultMsgID              = 0
	udpProtocolID       int = 0x41727101980
	udpConnectRequest         = 0
	udpAnnounceRequest        = 1
	udpScrapeRequest = 2
	udpError       = 3
	udpNoneEvent = 0

	incomingMsg    = 1
	outgoingMsg    = -1
)

var (
	keepALiveMsgLen        int = 0
	BitFieldMsgLen         int = 1
	chokeMsgLen            int = 1
	unChokeMsgLen          int = 1
	interestedMsgLen       int = 1
	UninterestedMsgLen     int = 1
	haveMsgLen             int = 5
	requestMsgLen          int = 13
	cancelMsgLen           int = 13
	pieceLen               int = 9
	portMsgLen                 = []byte{0, 0, 0, 3}
	HandShakePrefixLength      = []byte{19}
	ProtocolIdentifier         = []byte("BitTorrent app")
	BitTorrentReservedByte     = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	MaxMsgSize                 = 2000
	maxPiece               byte
)


type BaseMSG interface {
	handle()
	buildMsg(data []byte)
}


type MSG struct {
	ID             int
	Length         int
	MsgPrefixLen   int
	PieceIndex     int
	BeginIndex     int
	BitfieldRaw    []byte
	PieceLen       int
	Piece          []byte
	RawMsg         []byte
	availablePiece []bool
	InfoHash       []byte
	MyPeerID       string
	Peer           *Peer
	priority       int
	msgType        int
}

func (msg MSG) handleMsg()  {
	fmt.Printf("not implemented yet, msg id: %v", msg.ID)
}

type HanShakeMSG struct{
	BaseMSG
}

type ChockedMSg struct{
	BaseMSG
}

type UnChockedMSg struct{
	BaseMSG
}

type InterestedMSG struct{
	BaseMSG
}

type UnInterestedMSG struct {
	BaseMSG
}

type HaveMSG struct {
	BaseMSG
	PieceIndex int
}

type BitFieldMSG struct {
	BaseMSG
	Bitfield   []byte
}



type RequestMSG struct {
	BaseMSG
	PieceIndex int
	BeginIndex int
	Length int
}

type CancelRequestMSG struct {
	BaseMSG
	PieceIndex int
	BeginIndex int
	Length int
}

type PieceMSG struct {
	BaseMSG
	PieceIndex int
	BeginIndex int
	data    []byte
}




type HandShakeMsg struct {
	MsgID              int
	MsgLen             int
	reservedBytes      []byte
	InfoHash           []byte
	peerIp             string
	peerPort           string
	peerID             string
	rawMsg             []byte
	protocolIdentifier []byte
}

type UdpMSG struct {
	action        int
	connectionID  int
	transactionID int
	infoHash       []byte
	peerID         []byte
	downloaded     int
	left           int
	uploaded       int
	event          int
	ip             []byte
	key            int
	numWant        int
	port           int
	Interval       int
	leechers       int
	seeders        int
	PeersAddresses []byte
	
}

func GetMsg(msg MSG, peer *Peer) *MSG {
	var msgByte = make([]byte, 0)
	msgStruct := new(MSG)
	msgStruct.Peer = peer
	msgStruct.msgType = outgoingMsg
	switch msg.ID {

	case HandShakeMsgID:
		infoHashByte := msg.InfoHash
		peerIDByte := []byte(msg.MyPeerID)

		msgByte = bytes.Join([][]byte{HandShakePrefixLength, ProtocolIdentifier, BitTorrentReservedByte, infoHashByte, peerIDByte}, []byte(""))
		msgStruct.priority = JobQueue.HighPriority
	case RequestMsg:

		msgByte = bytes.Join([][]byte{intToByte(requestMsgLen, 4), intToByte(msg.ID, 1), intToByte(int(msg.PieceIndex), 4), intToByte(msg.BeginIndex, 4), intToByte(int(msg.PieceLen), 4)}, []byte(""))

		msgStruct.priority = 1

	case CancelMsg:
		msgByte = bytes.Join([][]byte{intToByte(cancelMsgLen, 4), intToByte(msg.ID, 1), intToByte(int(msg.PieceIndex), 4), intToByte(msg.BeginIndex, 4), intToByte(msg.PieceLen, 4)}, []byte(""))
		msgStruct.priority = JobQueue.HighPriority

	case PieceMsg:
		msgByte = bytes.Join([][]byte{intToByte(pieceLen+msg.PieceLen, 4), intToByte(msg.ID, 1), intToByte(msg.PieceIndex, 4), intToByte(msg.BeginIndex, 4), msg.Piece}, []byte(""))
		msgStruct.priority = JobQueue.HighPriority

	case HaveMsg:
		msgByte = bytes.Join([][]byte{intToByte(5, 4), intToByte(int(msg.ID), 1), intToByte(int(msg.PieceIndex), 4)}, []byte(""))
		msgStruct.priority = JobQueue.HighPriority
	default:

		msgByte = bytes.Join([][]byte{intToByte(1, 4), intToByte(msg.ID, 1)}, []byte(""))
		msgStruct.priority = JobQueue.HighPriority

	}

	msgStruct.RawMsg = msgByte
	return msgStruct
}

func ParseMsg(msg []byte, peer *Peer) (BaseMSG, error) {
	msgStruct := MSG{}
	msgStruct.msgType = incomingMsg
	msgStruct.Peer = peer

	var err error

	if len(msg) >= 5{

	msgStruct.Length =  int(binary.BigEndian.Uint32(msg[0:4]))
	id, _ := binary.Uvarint(msg[4:5])

	msgStruct.ID = int(id)

	msgStruct.priority = JobQueue.HighPriority
	}else{
		return nil, errors.New("msg is too short")
	}
	//fmt.Printf("BlockLength %v %v ID %v \n", msgStruct.BlockLength, binary.BigEndian.Uint32(msg[0:4]), msgStruct.ID)

	/*
	if msgStruct.BlockLength <= len(msg) {
		switch msgStruct.ID {
		case BitfieldMsg:
			msgStruct.PieceLen = int(math.Abs(float64(msgStruct.BlockLength - BitFieldMsgLen)))
			bitFieldMsg := BitFieldMSG{}
			bitFieldMsg.MSG = *msgStruct
			return bitFieldMsg, err
			bitFieldMsg.Bitfield = msg[5 : msgStruct.PieceLen+5]
			msgStruct.BitfieldRaw = msg[5 : msgStruct.PieceLen+5]
			msgStruct.priority = JobQueue.HighPriority
		case RequestMsg:
			msgStruct.PieceIndex = int(binary.BigEndian.Uint32(msg[5:9]))
			msgStruct.BeginIndex = int(binary.BigEndian.Uint32(msg[9:13]))
			msgStruct.PieceLen = int(binary.BigEndian.Uint32(msg[13:17]))
			msgStruct.priority = JobQueue.HighPriority

		case PieceMsg:
			msgStruct.PieceIndex = int(binary.BigEndian.Uint32(msg[5:9]))
			msgStruct.BeginIndex = int(binary.BigEndian.Uint32(msg[9:13]))
			msgStruct.PieceLen = int(math.Abs(float64(msgStruct.BlockLength - pieceLen)))
			msgStruct.Piece = make([]byte, msgStruct.PieceLen)
			//fmt.Printf("raw msg len %v, piece len %v msg len %v\n nine in uint32 %v\n", msg[0:4], msgStruct.PieceLen, msgStruct.BlockLength, pieceLen)
			end := msgStruct.PieceLen + 13


			copy(msgStruct.Piece, msg[13:end])
			msgStruct.priority = JobQueue.HighPriority

		case HaveMsg:
			msgStruct.PieceIndex = int(binary.BigEndian.Uint32(msg[5:9]))
			msgStruct.priority = JobQueue.HighPriority

		case CancelMsg:
			msgStruct.PieceIndex = int(binary.BigEndian.Uint32(msg[5:9]))
			msgStruct.BeginIndex = int(binary.BigEndian.Uint32(msg[9:13]))
			msgStruct.PieceLen = int(binary.BigEndian.Uint32(msg[13:17]))
			msgStruct.priority = JobQueue.HighPriority
		case UnchockeMsg:
			println("msgStruct.ID unchoking")
			println(msgStruct.ID)
		case ChockedMsg:

			println("msgStruct.ID choking")
			println(msgStruct.ID)

			// default msg -> interested,choke
		}
	} else {
		// TODO provides more accurate explanation
		// I will have to be more accurate later
		err = errors.New("wrong msg len")
	}
*/
	return nil, err
}

func ParseHandShake(msg []byte, infoHash string) (HandShakeMsg, error) {
	msgStruct := HandShakeMsg{}
	var err error = nil
	//println("new Handshake")
	//println(msg)

	if string(msg[0:1]) == string(HandShakePrefixLength) {
		if string(msg[1:20]) == string(ProtocolIdentifier) {
			if string(msg[28:48]) == infoHash {

			} else {
				err = errors.New("handshake \n bad info hash")

				//fmt.Printf("%v\n", err)
			}

		} else {
			err = errors.New("handshake \n bad app identifier")

			//fmt.Printf("%v\n", err)
		}
	} else {
		err = errors.New("handshake \n wrong prefix length")

		//fmt.Printf("%v\n", err)
	}

	if err == nil {
		MsgLen64, _ := binary.Uvarint(msg[0:1])

		msgStruct.MsgLen = int(MsgLen64)
		msgStruct.MsgID = HandShakeMsgID
		msgStruct.InfoHash = msg[28:48]
		ProtocolIdentifier = msg[1:20]
	}

	return msgStruct, err
}

func udpTrackerConnectMsg(msg UdpMSG) []byte {
	var msgByte = make([]byte, 0)

	msgByte = bytes.Join([][]byte{intToByte(msg.connectionID, 8), intToByte(msg.action, 4), intToByte(int(msg.transactionID), 4)}, []byte{})

	return msgByte

}

func udpTrackerAnnounceMsg(msg UdpMSG) []byte {
	var msgByte = make([]byte, 0)

	//TODO
	// make the intToByte method do this
	// it's cleaner

	msgByte = bytes.Join([][]byte{intToByte(msg.connectionID, 8), intToByte(msg.action, 4), intToByte(int(msg.transactionID), 4),msg.infoHash,msg.peerID,intToByte(msg.downloaded, 8),intToByte(msg.left, 8),intToByte(msg.uploaded, 8),intToByte(msg.event, 4),msg.ip,intToByte(msg.key, 4),intToByte(msg.numWant, 4),intToByte(msg.port, 2) }, []byte{})
	return msgByte
}

func parseUdpTrackerResponse(msg []byte,msgSize int) (UdpMSG,error){
	msgStruct := UdpMSG{}
	var err error

	if msgSize >= 16 {
		msgStruct.action = int(binary.BigEndian.Uint32(msg[0:4]))
		msgStruct.transactionID = int(binary.BigEndian.Uint32(msg[4:8]))

		if msgStruct.action == udpConnectRequest{
			msgStruct.connectionID = int(binary.BigEndian.Uint64(msg[8:16]))
		}else if msgStruct.action == udpAnnounceRequest{
			if msgSize > 20 {
				msgStruct.Interval = int(binary.BigEndian.Uint32(msg[8:12]))
				msgStruct.leechers = int(binary.BigEndian.Uint32(msg[12:16]))
				msgStruct.seeders = int(binary.BigEndian.Uint32(msg[16:20]))
				msgStruct.PeersAddresses = msg[20:msgSize]
			}else{
				err = errors.New("udp msg er | length too short 2")

			}
		}
	}else {
		err = errors.New("udp msg er | length too short 1 ")
	}
	return msgStruct,err
}



func isInBound(msg []byte,start,end int)error {
	if start < len(msg) || end > len(msg) {
		return errors.New("index is outOfBound")
	}

	return nil
}
