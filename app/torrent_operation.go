package app

import "DawnTorrent/protocol"

type pieceAvailabilityUpdate struct {
	piece *Piece
	action int
	peer protocol.PeerI
	fixQueue updatePiecePriority
}

func (operation pieceAvailabilityUpdate) Execute()  {
	operation.piece.updateAvailability(operation.action,operation.peer)
	if operation.piece.QueueIndex > -1 {
		operation.fixQueue(operation.piece.QueueIndex)
	}
}