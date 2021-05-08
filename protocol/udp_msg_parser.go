package protocol

import (
	"DawnTorrent/utils"
	"bytes"
	"encoding/binary"
)

const (
	udpProtocolId       = 0x41727101980
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
	return bytes.Join([][]byte{utils.IntToByte(udpProtocolId, 8), utils.IntToByte(udpConnectRequest, 4), utils.IntToByte(transactionId, 4)}, []byte{})
}

func newUdpAnnounceRequest(msg announceUdpMsg) []byte {

	return bytes.Join([][]byte{
		utils.IntToByte(msg.connectionID, 8),
		utils.IntToByte(msg.action, 4),
		utils.IntToByte(msg.transactionID, 4),
		[]byte(msg.infohash),
		[]byte(msg.peerId),
		utils.IntToByte(msg.downloaded, 8),
		utils.IntToByte(msg.left, 8),
		utils.IntToByte(msg.uploaded, 8),
		utils.IntToByte(msg.event, 4),
		utils.IntToByte(msg.ip, 4),
		utils.IntToByte(msg.key, 4),
		utils.IntToByte(msg.numWant, 4),
		utils.IntToByte(msg.port, 2),
	}, []byte{})
}

func parseUdpConnectionResponse(msg []byte) baseUdpMsg {

	return baseUdpMsg{
		action:        int(binary.BigEndian.Uint32(msg[0:4])),
		transactionID: int(binary.BigEndian.Uint32(msg[4:8])),
		connectionID:  int(binary.BigEndian.Uint64(msg[8:16])),
	}
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
	}

	if msgSize >= 26{
		announceResponse.peersAddresses = msg[26:msgSize]
	}

	print("msg size \n")
	print(msgSize)
	print("\n")

	//fmt.Printf("raw msg :\n %v",msg)

	return announceResponse
}
