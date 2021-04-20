package protocol

import (
	"encoding/binary"
	"errors"
)

func ParseMsg(msg []byte, peer *Peer) (BaseMSG, error) {
	baseMsg := MSG{}

	if len(msg) < 5 {
		return nil, errors.New("msg is too short")
	}
		baseMsg.Length =  int(binary.BigEndian.Uint32(msg[0:4]))
		id, _ := binary.Uvarint(msg[4:5])
		baseMsg.ID = int(id)

		switch baseMsg.ID {
		case BitfieldMsg:
			return  newBitfieldMSG(msg, baseMsg)
		case RequestMsg:
			return newRequestMSG(msg,baseMsg)

		case PieceMsg:
			return newPieceMSG(msg,baseMsg)
		case HaveMsg:
			return newHaveMSG(msg,baseMsg)

		case CancelMsg:
			return newCancelRequestMSG(msg,baseMsg)
		case UnchockeMsg:
			return newUnChockedMSg(msg,baseMsg)
		case ChockedMsg:
			return newChockedMSg(msg,baseMsg)
		case InterestedMsg:
			return newInterestedMSG(msg,baseMsg)
		case UnInterestedMsg:
			return newUnInterestedMSG(msg,baseMsg)
		}

	return nil, nil
}