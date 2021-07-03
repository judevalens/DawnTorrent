package tracker

import (
	"DawnTorrent/utils"
	"bytes"
	"encoding/binary"
)

const (
	udpProtocolId      = 0x41727101980
	udpConnectRequest  = 0
	UdpAnnounceRequest = 1
	udpScrapeRequest   = 2
	udpError           = 3
	udpNoneEvent       = 0
)

type BaseUdpMsg struct {
	Action        int
	ConnectionID  int
	TransactionID int
}

type AnnounceUdpMsg struct {
	BaseUdpMsg
	Infohash   string
	PeerId     string
	Downloaded int
	Uploaded   int
	Left       int
	Event      int
	Key        int
	NumWant    int
	Port       int
	Ip         int
}

type AnnounceResponseUdpMsg struct {
	BaseUdpMsg
	Interval       int
	NLeechers      int
	NSeeders       int
	PeersAddresses []byte
}

func NewUdpConnectionRequest(transactionId int) []byte {
	return bytes.Join([][]byte{utils.IntToByte(udpProtocolId, 8), utils.IntToByte(udpConnectRequest, 4), utils.IntToByte(transactionId, 4)}, []byte{})
}

func NewUdpAnnounceRequest(msg AnnounceUdpMsg) []byte {

	return bytes.Join([][]byte{
		utils.IntToByte(msg.ConnectionID, 8),
		utils.IntToByte(msg.Action, 4),
		utils.IntToByte(msg.TransactionID, 4),
		[]byte(msg.Infohash),
		[]byte(msg.PeerId),
		utils.IntToByte(msg.Downloaded, 8),
		utils.IntToByte(msg.Left, 8),
		utils.IntToByte(msg.Uploaded, 8),
		utils.IntToByte(msg.Event, 4),
		utils.IntToByte(msg.Ip, 4),
		utils.IntToByte(msg.Key, 4),
		utils.IntToByte(msg.NumWant, 4),
		utils.IntToByte(msg.Port, 2),
	}, []byte{})
}

func ParseUdpConnectionResponse(msg []byte) BaseUdpMsg {

	return BaseUdpMsg{
		Action:        int(binary.BigEndian.Uint32(msg[0:4])),
		TransactionID: int(binary.BigEndian.Uint32(msg[4:8])),
		ConnectionID:  int(binary.BigEndian.Uint64(msg[8:16])),
	}
}

func ParseAnnounceResponseUdpMsg(msg []byte, msgSize int) AnnounceResponseUdpMsg {
	announceResponse := AnnounceResponseUdpMsg{
		BaseUdpMsg: BaseUdpMsg{
			Action:        int(binary.BigEndian.Uint32(msg[0:4])),
			TransactionID: int(binary.BigEndian.Uint32(msg[4:8])),
		},
		Interval:  int(binary.BigEndian.Uint32(msg[8:12])),
		NLeechers: int(binary.BigEndian.Uint32(msg[12:16])),
		NSeeders:  int(binary.BigEndian.Uint32(msg[16:20])),
	}

	if msgSize >= 26{
		announceResponse.PeersAddresses = msg[26:msgSize]
	}

	print("msg size \n")
	print(msgSize)
	print("\n")

	//fmt.Printf("raw msg :\n %v",msg)

	return announceResponse
}
