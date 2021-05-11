package app

import (
	"DawnTorrent/protocol"
	"context"
)

type updatePieceAvailability struct {
	pieces []*Piece
	peer	protocol.PeerI
	action 	int
}


func (operation updatePieceAvailability) Execute(ctx context.Context)  {

	for i,piece := range operation.pieces{
		if operation.peer.HasPiece(i){
				if operation.action == 1{
					piece.Availability++
					piece.owners[operation.peer.GetId()] = true

				}
		}

	}
}