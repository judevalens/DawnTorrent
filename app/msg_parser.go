package app

import (
	"encoding/binary"
	"math"
)

func parseBitfieldMsg(rawMsg []byte, baseMSg header) (BitfieldMsg, error) {
	//TODO need to check for error
	msg := BitfieldMsg{}
	msg.TorrentMsg = baseMSg
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
	msg.TorrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	return msg, nil
}
func parseRequestMsg(rawMsg []byte, baseMSg header) (RequestMsg, error) {
	msg := RequestMsg{}
	msg.TorrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	msg.BeginIndex = int(binary.BigEndian.Uint32(rawMsg[9:13]))
	msg.BlockLength = int(binary.BigEndian.Uint32(rawMsg[13:17]))
	return msg, nil
}
func parseCancelRequestMsg(rawMsg []byte, baseMSg header) (CancelRequestMsg, error) {
	msg := CancelRequestMsg{}
	msg.TorrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	msg.BeginIndex = int(binary.BigEndian.Uint32(rawMsg[9:13]))
	msg.BlockLength = int(binary.BigEndian.Uint32(rawMsg[13:17]))
	return msg, nil
}
func parsePieceMsg(rawMsg []byte, baseMSg header) (PieceMsg, error) {
	msg := PieceMsg{}
	msg.TorrentMsg = baseMSg
	msg.PieceIndex = int(binary.BigEndian.Uint32(rawMsg[5:9]))
	msg.BeginIndex = int(binary.BigEndian.Uint32(rawMsg[9:13]))
	msg.BlockLength = int(math.Abs(float64(baseMSg.Length - pieceMsgLen)))
	msg.Payload = make([]byte, msg.BlockLength)
	copy(msg.Payload,rawMsg[13:msg.BlockLength+13])
	return msg,nil
}

func ParseMsg(msg []byte, peer *Peer) (TorrentMsg, error) {
	baseMsg := header{}
	baseMsg.peer = peer

	baseMsg.Length = int(binary.BigEndian.Uint32(msg[0:4]))
	// checking for keep-alive msg
	if baseMsg.Length == 0{
		return baseMsg,nil
	}

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
	default:
		return baseMsg,nil
	}

}
