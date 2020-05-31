package PeerProtocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	//	"strconv"
	//"encoding/binary"
)

const (
	HandShakeMsgID  = 19
	ChockedMsg      = 0
	UnchockeMsg     = 1
	InterestedMsg   = 2
	UninterestedMsg = 3
	HaveMsg         = 4
	BitfieldMsg     = 5
	RequestMsg      = 6
	PieceMsg        = 7
	CancelMsg       = 8
	DefaultMsgID    = 0
)

var (
	keepALiveMsgLen        = []byte{0, 0, 0, 0}
	BitFieldMsgLen         = []byte{0, 0, 0, 1}
	chokeMsgLen            = []byte{0, 0, 0, 1}
	unChokeMsgLen          = []byte{0, 0, 0, 1}
	interestedMsgLen       = []byte{0, 0, 0, 1}
	UninterestedMsgLen     = []byte{0, 0, 0, 1}
	haveMsgLen             = []byte{0, 0, 0, 5}
	requestMsgLen          = []byte{0, 0, 1, 3}
	cancelMsgLen           = []byte{0, 0, 1, 3}
	pieceLen               = 9
	portMsgLen             = []byte{0, 0, 0, 3}
	HandShakePrefixLength  = []byte{19}
	ProtocolIdentifier     = []byte("BitTorrent protocol")
	BitTorrentReservedByte = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	MaxMsgSize             = 2000
	maxPiece               byte
)

type MSG struct {
	MsgID          uint64
	MsgLen         uint32
	field          map[string][]byte
	PieceIndex     uint32
	BeginIndex     uint32
	BitfieldRaw    []byte
	PieceLen       uint32
	Piece          []byte
	rawMsg         []byte
	availablePiece []bool
	InfoHash       []byte
	MyPeerID       string
	Peer           *Peer
	priority       int
	msgType        int
}

type HandShakeMsg struct {
	MsgID              uint64
	MsgLen             uint64
	reservedBytes      []byte
	InfoHash           []byte
	peerIp             string
	peerPort           string
	peerID             string
	rawMsg             []byte
	protocolIdentifier []byte
}

func GetMsg(msg MSG, peer *Peer) MSG {
	var msgByte = make([]byte, 0)
	msgStruct := MSG{}
	msgStruct.Peer = peer
	msgStruct.msgType = outgoingMsg
	msgStruct.MsgID = msg.MsgID
	msgStruct.rawMsg = make([]byte, 0)
	switch msg.MsgID {

	case HandShakeMsgID:
		infoHashByte := []byte(msg.InfoHash)
		peerIDByte := []byte(msg.MyPeerID)

		msgByte = bytes.Join([][]byte{HandShakePrefixLength, ProtocolIdentifier, BitTorrentReservedByte, infoHashByte, peerIDByte}, []byte(""))
		msgStruct.priority = priority1
	case RequestMsg:

		msgByte = bytes.Join([][]byte{intToByte(int(13), 4), intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4), intToByte(int(msg.BeginIndex), 4), intToByte(int(msg.PieceLen), 4)}, []byte(""))

		msgStruct.priority = 1

	case CancelMsg:
		msgByte = bytes.Join([][]byte{requestMsgLen, intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4), intToByte(int(msg.BeginIndex), 4), intToByte(int(msg.PieceLen), 4)}, []byte(""))
		msgStruct.priority = priority1

	case PieceMsg:
		msgByte = bytes.Join([][]byte{intToByte(int(13+msg.PieceLen), 4), intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4), intToByte(int(msg.BeginIndex), 4), msg.Piece}, []byte(""))
		msgStruct.priority = priority1

	case HaveMsg:
		msgByte = bytes.Join([][]byte{intToByte(5, 4), intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4)}, []byte(""))
		msgStruct.priority = priority2
	default:

		msgByte = bytes.Join([][]byte{chokeMsgLen, intToByte(int(msg.MsgID), 1)}, []byte(""))
		msgStruct.priority = priority1

	}

	msgStruct.rawMsg = msgByte
	return msgStruct
}

func ParseMsg(msg []byte, peer *Peer) (MSG, error) {
	msgStruct := MSG{}
	msgStruct.msgType = incomingMsg
	msgStruct.Peer = peer

	var err error

	msgStruct.MsgLen = binary.BigEndian.Uint32(msg[0:4])
	id, _ := binary.Uvarint(msg[4:5])

	msgStruct.MsgID = id

	msgStruct.priority = priority2

	//fmt.Printf("MsgLen %v MsgID %v ", msgStruct.MsgLen, msgStruct.MsgID)

	if int(msgStruct.MsgLen) <= len(msg) {
		switch msgStruct.MsgID {
		case BitfieldMsg:
			msgStruct.PieceLen = uint32(math.Abs(float64(msgStruct.MsgLen - binary.BigEndian.Uint32(BitFieldMsgLen))))
			//	fmt.Printf("piece Len %v\n", msgStruct.PieceLen)
			msgStruct.BitfieldRaw = msg[5 : msgStruct.PieceLen+5]
			msgStruct.priority = priority2
		case RequestMsg:
			msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
			msgStruct.BeginIndex = binary.BigEndian.Uint32(msg[9:13])
			msgStruct.PieceLen = binary.BigEndian.Uint32(msg[13:17])
			msgStruct.priority = priority2

		case PieceMsg:
			msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
			msgStruct.BeginIndex = binary.BigEndian.Uint32(msg[9:13])
			msgStruct.PieceLen = uint32(math.Abs(float64(msgStruct.MsgLen - uint32(pieceLen))))
			///fmt.Printf("msg Header %v,msg Len %v , piecelen %v",msg[0:13], msgStruct.MsgLen, msgStruct.PieceLen)
			//os.Exit(1223)
			msgStruct.Piece = make([]byte,msgStruct.PieceLen)
			copy(msgStruct.Piece,msg[13 : msgStruct.PieceLen+13])
			msgStruct.priority = priority2

		case HaveMsg:
			msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
			msgStruct.priority = priority2

		case CancelMsg:
			msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
			msgStruct.BeginIndex = binary.BigEndian.Uint32(msg[9:13])
			msgStruct.PieceLen = binary.BigEndian.Uint32(msg[13:17])
			msgStruct.priority = priority2

			// default msg -> interested,choke
		}
	} else {
		// TODO provides more accurate explanation
		// I will have to be more accurate later
		err = errors.New("wrong msg len")
	}

	return msgStruct, err
}

func ParseHandShake(msg []byte, infoHash string) (HandShakeMsg, error) {
	msgStruct := HandShakeMsg{}
	var err error = nil
	println("new Handshake")
	println(msg)

	if string(msg[0:1]) == string(HandShakePrefixLength) {
		if string(msg[1:20]) == string(ProtocolIdentifier) {
			if string(msg[28:48]) == infoHash {

			} else {
				err = errors.New("handshake \n bad info hash")

				fmt.Printf("%v\n", err)
			}

		} else {
			err = errors.New("handshake \n bad protocol identifier")

			fmt.Printf("%v\n", err)
		}
	} else {
		err = errors.New("handshake \n wrong prefix length")

		fmt.Printf("%v\n", err)
	}

	if err == nil {
		msgStruct.MsgLen, _ = binary.Uvarint(msg[0:1])
		msgStruct.MsgID = HandShakeMsgID
		msgStruct.InfoHash = msg[28:48]
		ProtocolIdentifier = msg[1:20]
	}

	return msgStruct, err
}

func intToByte(n int, nByte int) []byte {
	b := make([]byte, nByte)

	if nByte < 2 {
		binary.PutUvarint(b, uint64(n))
	} else if nByte >= 2 && nByte < 4 {
		binary.BigEndian.PutUint16(b, uint16(n))
	} else if nByte >= 4 && nByte < 8 {
		binary.BigEndian.PutUint32(b, uint32(n))
	}

	return b

}
