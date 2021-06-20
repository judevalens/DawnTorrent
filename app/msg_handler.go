package app

type msgHandler struct {

}


func (manager *TorrentManager) HandleUnInterestedMsg(msg UnInterestedMsg) {
	manager.PeerManager.activePeers[msg.GetPeer().GetId()].isInterested = false
}

func (manager *TorrentManager) HandleInterestedMsg(msg InterestedMsg) {
	msg.GetPeer().isInterested = false
}

func (manager *TorrentManager) HandleUnChokeMsg(msg UnChockedMsg) {
	msg.GetPeer().SetChoke(false)

}
func (manager *TorrentManager) HandleChokeMsg(msg ChockedMSg) {
	msg.GetPeer().SetChoke(true)
}

func (manager *TorrentManager) HandleHaveMsg(msg HaveMsg) {
	msg.GetPeer().UpdateBitfield(msg.PieceIndex)
	// we use the sync channel because we can't update the piece priority pendingRequest concurrently
	manager.SyncOperation <- func() {
		manager.downloader.updatePiecePriority(msg.GetPeer())
	}
}

func (manager *TorrentManager) HandleBitFieldMsg(msg BitfieldMsg) {

	peer := msg.GetPeer()
	peer.SetBitField(msg.Bitfield)

	manager.SyncOperation <- func() {
		manager.downloader.updatePiecePriority(peer)
	}

}

func (manager *TorrentManager) HandlePieceMsg(msg PieceMsg) {
	//os.Exit(29)

	manager.downloader.PutPiece(msg)

}

func (manager *TorrentManager) HandleRequestMsg(msg RequestMsg) {
}

func (manager *TorrentManager) HandleCancelMsg(msg CancelRequestMsg) {
}
