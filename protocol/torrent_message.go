package protocol

import (
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
	ProtocolIdentifier         = []byte("BitTorrent protocol")
	BitTorrentReservedByte     = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	MaxMsgSize                 = 2000
	maxPiece               byte
)


type BaseMSG interface {
	handle()
}


type MSG struct {
	ID             int
	Length         int
}

func (msg MSG) handle() {
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

func newBitfieldMSG(data []byte, baseMSg BaseMSG) (BitfieldMSG,error){
	return BitfieldMSG{},nil
}


func newChockedMSg(data []byte, baseMSg BaseMSG) (ChockedMSg,error){
	return ChockedMSg{},nil
}

func newUnChockedMSg(data []byte, baseMSg BaseMSG) (UnChockedMSg,error){
	return UnChockedMSg{},nil
}
func newInterestedMSG(data []byte, baseMSg BaseMSG) (InterestedMSG,error){
	return InterestedMSG{},nil
}

func newUnInterestedMSG(data []byte, baseMSg BaseMSG) (UnInterestedMSG,error){
	return UnInterestedMSG{},nil
}

func newHaveMSG(data []byte, baseMSg BaseMSG) (HaveMSG,error){
	return HaveMSG{},nil
}
func newRequestMSG(data []byte, baseMSg BaseMSG) (RequestMSG,error)  {
	return RequestMSG{},nil
}
func newCancelRequestMSG(data []byte, baseMSg BaseMSG) (CancelRequestMSG,error)  {
	return CancelRequestMSG{},nil
}
func newPieceMSG(data []byte, baseMSg BaseMSG) (PieceMSG,error)  {
	return PieceMSG{},nil
}