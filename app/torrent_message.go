package app

import (
	"bytes"
	"errors"
	"log"
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

type TorrentMsg interface {
	getId() int
	handleMsg(manager *TorrentManager)
	Marshal() []byte
	GetPeer() *Peer
}

type header struct {
	ID     int
	Length int
	peer   *Peer
}

func (msg header) handleMsg(manager *TorrentManager) {
	log.Printf("this is a keep alive msg from: %v", msg.GetPeer().GetId())
}

func (msg header) GetPeer() *Peer {
	return msg.peer
}

func (msg header) Marshal() []byte {
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
	TorrentMsg
}

func (h HanShakeMsg) handle() {
	panic("implement me")
}

func (h HanShakeMsg) buildMsg(data []byte) {
	panic("implement me")
}

type ChockedMSg struct {
	TorrentMsg
}

func (msg ChockedMSg) handleMsg(manager *TorrentManager) {
	manager.HandleChokeMsg(msg)

}

type UnChockedMsg struct {
	TorrentMsg
}

func (msg UnChockedMsg) handleMsg(manager *TorrentManager) {
	manager.HandleUnChokeMsg(msg)

}

type InterestedMsg struct {
	TorrentMsg
}

func (msg InterestedMsg) handleMsg(manager *TorrentManager) {
	manager.HandleInterestedMsg(msg)

}

type UnInterestedMsg struct {
	TorrentMsg
}

func (msg UnInterestedMsg) handleMsg(manager *TorrentManager) {
	manager.HandleUnInterestedMsg(msg)

}

type HaveMsg struct {
	TorrentMsg
	PieceIndex int
}

func (msg HaveMsg) handleMsg(manager *TorrentManager) {
	manager.HandleHaveMsg(msg)

}

type BitfieldMsg struct {
	TorrentMsg
	Bitfield     []byte
	BitFieldSize int
}

func (msg BitfieldMsg) handleMsg(manager *TorrentManager) {
	manager.HandleBitFieldMsg(msg)

}

type RequestMsg struct {
	TorrentMsg
	PieceIndex  int
	BeginIndex  int
	BlockLength int
}

func (request RequestMsg) handleMsg(manager *TorrentManager) {
	manager.HandleRequestMsg(request)
}

type CancelRequestMsg struct {
	TorrentMsg
	PieceIndex  int
	BeginIndex  int
	BlockLength int
}

func (msg CancelRequestMsg) handleMsg(manager *TorrentManager) {
	manager.HandleCancelMsg(msg)
}

type PieceMsg struct {
	TorrentMsg
	PieceIndex  int
	BeginIndex  int
	BlockLength int
	Payload     []byte
}

func (msg PieceMsg) handleMsg(manager *TorrentManager) {
	manager.HandlePieceMsg(msg)
}
