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
	HandShakeMsg    = 10
	ChockeMsg       = 0
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
	pieceLen               = []byte{0, 0, 0, 9}
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
	PeerID 		int
}

func GetMsg(msg MSG) []byte {
	var msgByte = make([]byte, 0)
	switch msg.MsgID {

	case HandShakeMsg:
		infoHashByte := []byte(msg.InfoHash)
		peerIDByte := []byte(msg.MyPeerID)

		msgByte = bytes.Join([][]byte{HandShakePrefixLength, ProtocolIdentifier, BitTorrentReservedByte, infoHashByte, peerIDByte}, []byte(""))
	case RequestMsg:
		msgByte = bytes.Join([][]byte{requestMsgLen, intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4), intToByte(int(msg.BeginIndex), 4), intToByte(int(msg.PieceLen), 4)}, []byte(""))
	case CancelMsg:
		msgByte = bytes.Join([][]byte{requestMsgLen, intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4), intToByte(int(msg.BeginIndex), 4), intToByte(int(msg.PieceLen), 4)}, []byte(""))
	case PieceMsg:
		msgByte = bytes.Join([][]byte{intToByte(int(13+msg.PieceLen), 4), intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4), intToByte(int(msg.BeginIndex), 4), msg.Piece}, []byte(""))
	case HaveMsg:
		msgByte = bytes.Join([][]byte{intToByte(5, 4), intToByte(int(msg.MsgID), 1), intToByte(int(msg.PieceIndex), 4)}, []byte(""))
	default:
		idByte := make([]byte, 1)
		binary.PutUvarint(idByte, msg.MsgID)
		msgByte = bytes.Join([][]byte{chokeMsgLen, intToByte(int(msg.MsgID), 1)}, []byte(""))

	}

	fmt.Printf("%v\n", msgByte)
	hs := string(msgByte)
	fmt.Printf("%v\n", string(hs))

	return msgByte
}

func ParseMsg(msg []byte, torrent *Torrent) MSG {
	msgStruct := MSG{}

	msgStruct.MsgLen = binary.BigEndian.Uint32(msg[0:4])
	msgStruct.MsgID, _ = binary.Uvarint(msg[4:5])

	fmt.Printf("MsgLen %v MsgID %v ", msgStruct.MsgLen, msgStruct.MsgID)

	switch msgStruct.MsgID {
	case BitfieldMsg:
		msgStruct.PieceLen = uint32(math.Abs(float64(msgStruct.MsgLen - binary.BigEndian.Uint32(BitFieldMsgLen))))
		fmt.Printf("piece Len %v\n", msgStruct.PieceLen)
		msgStruct.BitfieldRaw = msg[5 : msgStruct.PieceLen+5]
	case RequestMsg:
		msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
		msgStruct.BeginIndex = binary.BigEndian.Uint32(msg[9:13])
		msgStruct.PieceLen = binary.BigEndian.Uint32(msg[13:17])
	case PieceMsg:
		msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
		msgStruct.BeginIndex = binary.BigEndian.Uint32(msg[9:13])
		msgStruct.PieceLen = uint32(math.Abs(float64(msgStruct.MsgLen - binary.BigEndian.Uint32(pieceLen))))
		msgStruct.Piece = msg[17 : msgStruct.PieceLen+17]
	case HaveMsg:
		msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
	case CancelMsg:
		msgStruct.PieceIndex = binary.BigEndian.Uint32(msg[5:9])
		msgStruct.BeginIndex = binary.BigEndian.Uint32(msg[9:13])
		msgStruct.PieceLen = binary.BigEndian.Uint32(msg[13:17])
	}

	return msgStruct
}

func ParseHandShake(msg []byte, infoHash string) (MSG, error) {
	msgStruct := MSG{}
	var err error = nil
	println("new Handshake")
	println(msg)
	if string(msg[0:1]) == string(HandShakePrefixLength) {
		if string(msg[1:20]) == string(ProtocolIdentifier) {
			fmt.Printf("info hash : %v\n", msg[28:48])
			if string(msg[28:48]) == infoHash {
				msgStruct.MsgID = HandShakeMsg
				msgStruct.rawMsg = msg
			} else {
				err = errors.New("handshake \n bad info hash")

				fmt.Printf("%v\n", err)
			}

		} else {
			err = errors.New("handshake \n wrong protocol")

			fmt.Printf("%v\n", err)
		}
	} else {
		err = errors.New("handshake \n wrong protocol")

		fmt.Printf("%v\n", err)
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
