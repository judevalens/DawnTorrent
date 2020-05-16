package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	//	"strconv"
	//"encoding/binary"
)

const (
	HanShakeMsg     = 10
	ChockeMsg       = 0
	UnchockeMsg     = 1
	InterestedMsg   = 2
	UninterestedMsg = 3
	HaveMsg         = 4
	BitfieldMsg     = 5
	RequestMsg      = 6
	PieceMsg        = 7
	CancelMsg       = 8
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
	msgID      uint32
	msgLen     uint32
	field      map[string][]byte
	pieceIndex uint32
	begin      uint32
	bitfield   []byte
	pieceLen   uint32
	piece      []byte
	rawMsg     []byte
}

func GetMsg(infoHash string, peerID string, msgType int) []byte {
	var msg = make([]byte, 0)
	switch msgType {

	case HanShakeMsg:
		infoHashByte := []byte(infoHash)
		peerIDByte := []byte(peerID)

		msg = bytes.Join([][]byte{HandShakePrefixLength, ProtocolIdentifier, BitTorrentReservedByte, infoHashByte, peerIDByte}, []byte(""))
	}

	fmt.Printf("%v\n", msg)
	hs := string(msg)
	fmt.Printf("%v\n", string(hs))

	return msg
}

func parseMsg(msg []byte) MSG {
	msgStruct := MSG{}
	if string(msg) == string(HandShakePrefixLength) {
		// It is a hanShake
		msgStruct.msgID = HanShakeMsg
		msgStruct.msgLen = binary.BigEndian.Uint32(msg[0:1])
	} else {
		msgStruct.msgLen = binary.BigEndian.Uint32(msg[0:4])
		msgStruct.msgID = binary.BigEndian.Uint32(msg[4:5])

		switch msgStruct.msgID {
		case BitfieldMsg:
			msgStruct.pieceLen = uint32(math.Abs(float64(msgStruct.pieceLen-binary.BigEndian.Uint32(BitFieldMsgLen))))
			msgStruct.bitfield = msg[5:msgStruct.msgLen+5]
		case RequestMsg:
			msgStruct.pieceIndex = binary.BigEndian.Uint32(msg[5:9])
			msgStruct.begin = binary.BigEndian.Uint32(msg[9:13])
			msgStruct.pieceLen = binary.BigEndian.Uint32(msg[13:17])
		case PieceMsg:
			msgStruct.pieceIndex = binary.BigEndian.Uint32(msg[5:9])
			msgStruct.begin = binary.BigEndian.Uint32(msg[9:13])
			msgStruct.pieceLen = uint32(math.Abs(float64(binary.BigEndian.Uint32(msg[13:17]) - binary.BigEndian.Uint32(pieceLen))))
			msgStruct.piece = msg[17:msgStruct.pieceLen]
		}
	}

	return msgStruct
}
