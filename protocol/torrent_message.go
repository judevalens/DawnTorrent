package protocol

import (
	"bytes"
	"errors"
	"fmt"
)


const (
	HandShakeMsgID         = 19
	ChockedMsg             = 0
	UnchockeMsg            = 1
	InterestedMsg          = 2
	UnInterestedMsg        = 3
	HaveMsg                = 4
	BitfieldMsg            = 5
	RequestMsg             = 6
	PieceMsg               = 7
	CancelMsg              = 8
	DefaultMsgID           = 0
	udpProtocolID      int = 0x41727101980
	udpConnectRequest      = 0
	udpAnnounceRequest     = 1
	udpScrapeRequest       = 2
	udpError       = 3
	udpNoneEvent = 0

	incomingMsg    = 1
	outgoingMsg    = -1
)

const (
	handShakeLen = 58
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
	HandShakePrefixLength      = 19
	BittorrentIdentifier       = "BitTorrent protocol"
	BitTorrentReservedByte     = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	MaxMsgSize                 = 2000
	maxPiece               byte
)


type BaseMSG interface {
	handleRequest()
	marshal() []byte
}


type MSG struct {
	ID             int
	Length         int
}

type HandShake struct {
	pstrlen string
	pstr string
	infoHash string
	reservedBytes []byte
	peerID string
}

func newHandShakeMsg(infohash string, peerID string) []byte  {
	msg := HandShake{
		infoHash: infohash,
	}
	return msg.marshal()
}

func(h HandShake) marshal() []byte {
	return bytes.Join([][]byte{{byte(HandShakePrefixLength)}, []byte(BittorrentIdentifier), BitTorrentReservedByte, []byte(h.infoHash), []byte(h.peerID)}, []byte(""))
}
func(h HandShake) handleRequest() {
}

func parseHandShake(data []byte)  (*HandShake,error){
	if len(data) < handShakeLen{
		return nil,errors.New("could not parse msg")
	}

	handShakeMsg := new(HandShake)

	handShakeMsg.pstrlen = string(data[0:1])
	handShakeMsg.pstr = string(data[1:len(BittorrentIdentifier)])
	handShakeMsg.reservedBytes = data[len(BittorrentIdentifier):len(BittorrentIdentifier)+8]
	handShakeMsg.reservedBytes = data[len(BittorrentIdentifier)+8:len(BittorrentIdentifier)+28]
	handShakeMsg.reservedBytes = data[len(BittorrentIdentifier)+28:len(BittorrentIdentifier)+48]
	return handShakeMsg,nil

}

func (msg MSG) handleRequest() {
	panic("implement me")
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

type BitfieldMSG struct {
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



func parseBitfieldMSG(data []byte, baseMSg BaseMSG) (BitfieldMSG,error){
	return BitfieldMSG{},nil
}


func parseChockedMSg(data []byte, baseMSg BaseMSG) (ChockedMSg,error){
	return ChockedMSg{},nil
}

func parseUnChockedMSg(data []byte, baseMSg BaseMSG) (UnChockedMSg,error){
	return UnChockedMSg{},nil
}
func parseInterestedMSG(data []byte, baseMSg BaseMSG) (InterestedMSG,error){
	return InterestedMSG{},nil
}

func parseUnInterestedMSG(data []byte, baseMSg BaseMSG) (UnInterestedMSG,error){
	return UnInterestedMSG{},nil
}

func parseHaveMSG(data []byte, baseMSg BaseMSG) (HaveMSG,error){
	return HaveMSG{},nil
}
func parseRequestMSG(data []byte, baseMSg BaseMSG) (RequestMSG,error)  {
	return RequestMSG{},nil
}
func parseCancelRequestMSG(data []byte, baseMSg BaseMSG) (CancelRequestMSG,error)  {
	return CancelRequestMSG{},nil
}
func parsePieceMSG(data []byte, baseMSg BaseMSG) (PieceMSG,error)  {
	return PieceMSG{},nil
}