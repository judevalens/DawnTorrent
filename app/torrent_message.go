package app

import (
	"DawnTorrent/protocol"
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

const (
	HandShakeMsgId       = 19
	ChokeMsgId           = 0
	UnChokeMsgId         = 1
	InterestedMsgId      = 2
	UnInterestedMsgId    = 3
	HaveMsgId            = 4
	BitfieldMsgId        = 5
	RequestMsgId         = 6
	PieceMsgId           = 7
	CancelMsgId          = 8
	BittorrentIdentifier = "BitTorrent protocol"
)

const (
	defaultMsgLen = 1
	requestMsgLen = 13
	cancelMsgLen = 13
	pieceMsgLen = 9
	haveMsgLen = 5
)

const (
	handShakeLen = 68
)

var (
	MaxMsgSize = 2000
	maxPiece   byte
)

type torrentMsg interface {
	getId() int
	handleMsg(manager *TorrentManager)
	marshal() []byte

}

type header struct {
	ID     int
	Length int
}


func (msg header) marshal() []byte {
	panic("implement me")
}

func (msg header) handleMsg(manager *TorrentManager) {
	panic("implement me")
}
func (msg header) getId() int{
	return msg.ID
}

type HandShake struct {
	pstrlen       string
	pstr          string
	infoHash      []byte
	reservedBytes []byte
	peerID        string
}

func newHandShakeMsg(infohash []byte, peerID string) []byte {
	msg := HandShake{
		infoHash: infohash,
		peerID: peerID,
	}
	return msg.marshal()
}

func (h HandShake) marshal() []byte {
	return bytes.Join([][]byte{
		{byte(HandShakeMsgId)},
		[]byte(BittorrentIdentifier),
		make([]byte, 8),
		h.infoHash,
		[]byte(h.peerID)},
		[]byte{})
}
func (h HandShake) handleMsg(*TorrentManager) {
}

func parseHandShake(data []byte) (*HandShake, error) {
	if len(data) < handShakeLen {
		return nil, errors.New("could not parse msg")
	}

	handShakeMsg := new(HandShake)

	handShakeMsg.pstrlen = string(data[0:1])
	handShakeMsg.pstr = string(data[1:20])
	handShakeMsg.reservedBytes = data[20 : 28]
	handShakeMsg.infoHash = data[28 : 48]
	handShakeMsg.peerID = string(data[48:68])
	return handShakeMsg, nil

}

type HanShakeMsg struct {
	torrentMsg
}

func (h HanShakeMsg) handle() {
	panic("implement me")
}

func (h HanShakeMsg) buildMsg(data []byte) {
	panic("implement me")
}

type ChockedMSg struct {
	torrentMsg
}

func (msg ChockedMSg) handleMsg(manager *TorrentManager) {
	manager.handleChokeMsg(msg)

}

type UnChockedMsg struct {
	torrentMsg
}

func (msg UnChockedMsg) handleMsg(manager *TorrentManager) {
	manager.handleUnChokeMsg(msg)

}

type InterestedMsg struct {
	torrentMsg
}

func (msg InterestedMsg) handleMsg(manager *TorrentManager) {
	manager.handleInterestedMsg(msg)

}

type UnInterestedMsg struct {
	torrentMsg
}

func (msg UnInterestedMsg) handleMsg(manager *TorrentManager) {
	manager.handleUnInterestedMsg(msg)

}

type HaveMsg struct {
	torrentMsg
	PieceIndex int
}

func (msg HaveMsg) handleMsg(manager *TorrentManager) {
	manager.handleHaveMsg(msg)

}

type BitfieldMsg struct {
	torrentMsg
	Bitfield     []byte
	BitFieldSize int
}

func (msg BitfieldMsg) handleMsg(manager *TorrentManager) {
	manager.handleBitFieldMsg(msg)

}

type RequestMsg struct {
	torrentMsg
	PieceIndex  int
	BeginIndex  int
	BlockLength int
}

func (request RequestMsg) handleMsg(manager *TorrentManager) {
	manager.handleRequestMsg(request)
}

type CancelRequestMsg struct {
	torrentMsg
	PieceIndex  int
	BeginIndex  int
	BlockLength int
}

func (msg CancelRequestMsg) handleMsg(manager *TorrentManager) {
	manager.handleCancelMsg(msg)
}

type PieceMsg struct {
	torrentMsg
	PieceIndex  int
	BeginIndex  int
	BlockLength int
	block       []byte
}

func (msg PieceMsg) handleMsg(manager *TorrentManager) {
	manager.handlePieceMsg(msg)
}

func parseBitfieldMsg(rawMsg []byte, baseMSg header) (BitfieldMsg, error) {
	//TODO need to check for error
	msg := BitfieldMsg{}
	msg.torrentMsg = baseMSg
	msg.BitFieldSize = int(math.Abs(float64(baseMSg.Length - defaultMsgLen)))
	msg.Bitfield = rawMsg[5 :msg.BitFieldSize+5]
	return msg, nil
}
func parseChockedMSg(data []byte, baseMSg header) (ChockedMSg, error) {
	//TODO need to check for error
	msg := ChockedMSg{baseMSg}
	return msg, nil
}
func parseUnChockedMSg(data []byte, baseMSg header) (UnChockedMsg, error) {
	//TODO need to check for error
	msg := UnChockedMsg{baseMSg}
	return msg, nil
}
func parseInterestedMsg(data []byte, baseMSg header) (InterestedMsg, error) {
	//TODO need to check for error
	msg := InterestedMsg{baseMSg}
	return msg, nil
}
func parseUnInterestedMsg(data []byte, baseMSg header) (UnInterestedMsg, error) {
	return UnInterestedMsg{baseMSg}, nil
}
func parseHaveMsg(rawMsg []byte, baseMSg header) (HaveMsg, error) {

	msg := HaveMsg{}
	msg.torrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	return msg, nil
}
func parseRequestMsg(rawMsg []byte, baseMSg header) (RequestMsg, error) {
	msg := RequestMsg{}
	msg.torrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	msg.BeginIndex = int(binary.BigEndian.Uint32(rawMsg[9:13]))
	msg.BlockLength = int(binary.BigEndian.Uint32(rawMsg[13:17]))
	return msg, nil
}
func parseCancelRequestMsg(rawMsg []byte, baseMSg header) (CancelRequestMsg, error) {
	msg := CancelRequestMsg{}
	msg.torrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	msg.BeginIndex = int(binary.BigEndian.Uint32(rawMsg[9:13]))
	msg.BlockLength = int(binary.BigEndian.Uint32(rawMsg[13:17]))
	return msg, nil
}
func parsePieceMsg(rawMsg []byte, baseMSg header) (PieceMsg, error) {
	msg := PieceMsg{}
	msg.torrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	msg.BeginIndex = int(binary.BigEndian.Uint32(rawMsg[9:13]))
	msg.BlockLength = int(math.Abs(float64(baseMSg.Length - pieceMsgLen)))
	msg.block = make([]byte, msg.BlockLength)

	return msg,nil
}

func ParseMsg(msg []byte, peer protocol.PeerI) (torrentMsg, error) {
	baseMsg := header{}
	if len(msg) < 5 {
		return nil, errors.New("msg is too short")
	}
	baseMsg.Length = int(binary.BigEndian.Uint32(msg[0:4]))
	id, _ := binary.Uvarint(msg[4:5])
	baseMsg.ID = int(id)

	switch baseMsg.ID {
	case BitfieldMsgId:
		return parseBitfieldMsg(msg, baseMsg)
	case RequestMsgId:
		return parseRequestMsg(msg, baseMsg)
	case PieceMsgId:
		return parsePieceMsg(msg, baseMsg)
	case HaveMsgId:
		return parseHaveMsg(msg, baseMsg)
	case CancelMsgId:
		return parseCancelRequestMsg(msg, baseMsg)
	case UnChokeMsgId:
		return parseUnChockedMSg(msg, baseMsg)
	case ChokeMsgId:
		return parseChockedMSg(msg, baseMsg)
	case InterestedMsgId:
		return parseInterestedMsg(msg, baseMsg)
	case UnInterestedMsgId:
		return parseUnInterestedMsg(msg, baseMsg)
	}

	return nil, nil
}