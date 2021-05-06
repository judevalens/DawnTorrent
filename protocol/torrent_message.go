package protocol

import (
	"DawnTorrent/utils"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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
	handShakeLen = 58
)

var (
	MaxMsgSize = 2000
	maxPiece   byte
)

type BaseMsg interface {
	handleMsg(manager *TorrentManager)
	marshal() []byte
}

type Msg struct {
	ID     int
	Length int
}

type HandShake struct {
	pstrlen       string
	pstr          string
	infoHash      string
	reservedBytes []byte
	peerID        string
}

func newHandShakeMsg(infohash string, peerID string) []byte {
	msg := HandShake{
		infoHash: infohash,
	}
	return msg.marshal()
}

func (h HandShake) marshal() []byte {
	return bytes.Join([][]byte{
		{byte(HandShakeMsgId)},
		[]byte(BittorrentIdentifier),
		make([]byte, 8),
		[]byte(h.infoHash),
		[]byte(h.peerID)},
		[]byte(""))
}
func (h HandShake) handleMsg(*TorrentManager) {
}

func parseHandShake(data []byte) (*HandShake, error) {
	if len(data) < handShakeLen {
		return nil, errors.New("could not parse msg")
	}

	handShakeMsg := new(HandShake)

	handShakeMsg.pstrlen = string(data[0:1])
	handShakeMsg.pstr = string(data[1:len(BittorrentIdentifier)])
	handShakeMsg.reservedBytes = data[len(BittorrentIdentifier) : len(BittorrentIdentifier)+8]
	handShakeMsg.reservedBytes = data[len(BittorrentIdentifier)+8 : len(BittorrentIdentifier)+28]
	handShakeMsg.reservedBytes = data[len(BittorrentIdentifier)+28 : len(BittorrentIdentifier)+48]
	return handShakeMsg, nil

}

func (msg Msg) handleRequest() {
	panic("implement me")
}

func (msg Msg) handleMsg() {
	fmt.Printf("not implemented yet, msg id: %v", msg.ID)
}

type HanShakeMsg struct {
	BaseMsg
}

func (h HanShakeMsg) handle() {
	panic("implement me")
}

func (h HanShakeMsg) buildMsg(data []byte) {
	panic("implement me")
}

type ChockedMSg struct {
	BaseMsg
}

func (msg ChockedMSg) handleMsg(manager *TorrentManager) {
	manager.handleChokeMsg(msg)

}

type UnChockedMsg struct {
	BaseMsg
}

func (msg UnChockedMsg) handleMsg(manager *TorrentManager) {
	manager.handleUnChokeMsg(msg)

}

type InterestedMsg struct {
	BaseMsg
}

func (msg InterestedMsg) handleMsg(manager *TorrentManager) {
	manager.handleInterestedMsg(msg)

}

type UnInterestedMsg struct {
	BaseMsg
}

func (msg UnInterestedMsg) handleMsg(manager *TorrentManager) {
	manager.handleUnInterestedMsg(msg)

}

type HaveMsg struct {
	BaseMsg
	PieceIndex int
}

func (msg HaveMsg) handleMsg(manager *TorrentManager) {
	manager.handleHaveMsg(msg)

}

type BitfieldMsg struct {
	BaseMsg
	Bitfield []byte
}

func (msg BitfieldMsg) handleMsg(manager *TorrentManager) {
	manager.handleBitFieldMsg(msg)

}

type RequestMsg struct {
	BaseMsg
	PieceIndex int
	BeginIndex int
	Length     int
}

func (msg RequestMsg) handleMsg(manager *TorrentManager) {
	manager.handleRequestMsg(msg)
}

type CancelRequestMsg struct {
	BaseMsg
	PieceIndex int
	BeginIndex int
	Length     int
}

func (msg CancelRequestMsg) handleMsg(manager *TorrentManager) {
	manager.handleCancelMsg(msg)
}

type PieceMsg struct {
	BaseMsg
	PieceIndex int
	BeginIndex int
	data       []byte
}

func (msg PieceMsg) handleMsg(manager *TorrentManager) {
	manager.handlePieceMsg(msg)
}

func parseBitfieldMsg(data []byte, baseMSg Msg) (BitfieldMsg, error) {
	return BitfieldMsg{}, nil
}

func parseChockedMSg(data []byte, baseMSg Msg) (ChockedMSg, error) {
	return ChockedMSg{}, nil
}

func parseUnChockedMSg(data []byte, baseMSg Msg) (UnChockedMsg, error) {
	return UnChockedMsg{}, nil
}
func parseInterestedMsg(data []byte, baseMSg Msg) (InterestedMsg, error) {
	return InterestedMsg{}, nil
}

func parseUnInterestedMsg(data []byte, baseMSg Msg) (UnInterestedMsg, error) {
	return UnInterestedMsg{}, nil
}

func parseHaveMsg(data []byte, baseMSg Msg) (HaveMsg, error) {
	return HaveMsg{}, nil
}
func parseRequestMsg(data []byte, baseMSg Msg) (RequestMsg, error) {
	return RequestMsg{}, nil
}
func parseCancelRequestMsg(data []byte, baseMSg Msg) (CancelRequestMsg, error) {
	return CancelRequestMsg{}, nil
}
func parsePieceMsg(data []byte, baseMSg Msg) (PieceMsg, error) {
	return PieceMsg{}, nil
}

func ParseMsg(msg []byte, peer *Peer) (BaseMsg, error) {
	baseMsg := Msg{}
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

const (
	udpProtocolId      int = 0x41727101980
	udpConnectRequest      = 0
	udpAnnounceRequest     = 1
	udpScrapeRequest       = 2
	udpError               = 3
	udpNoneEvent           = 0
)

type baseUdpMsg struct {
	action        int
	connectionID  int
	transactionID int
}

type announceUdpMsg struct {
	baseUdpMsg
	infohash   string
	peerId     string
	downloaded int
	uploaded   int
	left       int
	event      int
	key        int
	numWant    int
	port       int
	ip         int
}

type announceResponseUdpMsg struct {
	baseUdpMsg
	interval       int
	nLeechers      int
	nSeeders       int
	peersAddresses []byte
}

func newUdpConnectionRequest(transactionId int) []byte {
	return bytes.Join([][]byte{utils.IntToByte(udpProtocolId, 8), utils.IntToByte(udpConnectRequest, 4), utils.IntToByte(int(transactionId), 4)}, []byte{})
}

func newUdpAnnounceRequest(msg announceUdpMsg) []byte {

	return bytes.Join([][]byte{
		utils.IntToByte(msg.connectionID, 8),
		utils.IntToByte(msg.action, 4),
		utils.IntToByte(msg.transactionID, 4),
		[]byte(msg.infohash), []byte(msg.peerId),
		utils.IntToByte(msg.downloaded, 8),
		utils.IntToByte(msg.left, 8),
		utils.IntToByte(msg.uploaded, 8),
		utils.IntToByte(msg.event, 4),
		utils.IntToByte(msg.ip, 4), utils.IntToByte(msg.key, 4),
		utils.IntToByte(msg.numWant, 4),
		utils.IntToByte(msg.port, 2)}, []byte{})
}

func parseUdpConnectionResponse(msg []byte) baseUdpMsg {

	return baseUdpMsg{
		action:        int(binary.BigEndian.Uint32(msg[0:4])),
		transactionID: int(binary.BigEndian.Uint32(msg[4:8])),
		connectionID:  int(binary.BigEndian.Uint64(msg[8:16])),
	}
}

func parseUdpTrackerResponse(msg []byte, msgSize int) (UdpMSG, error) {
	msgStruct := UdpMSG{}
	var err error

	if msgSize >= 16 {
		msgStruct.action = int(binary.BigEndian.Uint32(msg[0:4]))
		msgStruct.transactionID = int(binary.BigEndian.Uint32(msg[4:8]))

		if msgStruct.action == udpConnectRequest {
			msgStruct.connectionID = int(binary.BigEndian.Uint64(msg[8:16]))
		} else if msgStruct.action == udpAnnounceRequest {
			if msgSize > 20 {
				msgStruct.Interval = int(binary.BigEndian.Uint32(msg[8:12]))
				msgStruct.leechers = int(binary.BigEndian.Uint32(msg[12:16]))
				msgStruct.seeders = int(binary.BigEndian.Uint32(msg[16:20]))
				msgStruct.PeersAddresses = msg[20:msgSize]
			} else {
				err = errors.New("udp msg er | length too short 2")

			}
		}
	} else {
		err = errors.New("udp msg er | length too short 1 ")
	}
	return msgStruct, err
}

func parseAnnounceResponseUdpMsg(msg []byte, msgSize int) announceResponseUdpMsg {
	announceResponse := announceResponseUdpMsg{
		baseUdpMsg: baseUdpMsg{
			action:        int(binary.BigEndian.Uint32(msg[0:4])),
			transactionID: int(binary.BigEndian.Uint32(msg[4:8])),
		},
		interval:       int(binary.BigEndian.Uint32(msg[8:12])),
		nLeechers:      int(binary.BigEndian.Uint32(msg[12:16])),
		nSeeders:       int(binary.BigEndian.Uint32(msg[16:20])),
		peersAddresses: msg[20:msgSize],
	}

	return announceResponse
}
