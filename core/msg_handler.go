package core

import "log"

type msgHandler struct {

}


func (manager *TorrentManager) HandleUnInterestedMsg(msg UnInterestedMsg) {
	 msg.GetPeer().isInterested = false
}

func (manager *TorrentManager) HandleInterestedMsg(msg InterestedMsg) {
	msg.GetPeer().isInterested = false
}

func (manager *TorrentManager) HandleUnChokeMsg(msg UnChockedMsg) {
	msg.GetPeer().SetChoke(false)
	manager.PeerManager.peerAlert.Signal()

}
func (manager *TorrentManager) HandleChokeMsg(msg ChockedMSg) {
	log.Printf("choking......")

	msg.GetPeer().SetChoke(true)
	manager.PeerManager.peerAlert.Signal()
}

func (manager *TorrentManager) HandleHaveMsg(msg HaveMsg) {
	msg.GetPeer().UpdateBitfield(msg.PieceIndex)
	manager.PeerManager.peerAlert.Signal()

	// we use the sync channel because we can't update the piece priority pendingRequest concurrently
	manager.downloader.updatePiecePriority(msg.GetPeer())
}

func (manager *TorrentManager) HandleBitFieldMsg(msg BitfieldMsg) {

	peer := msg.GetPeer()
	peer.SetBitField(msg.Bitfield)
	manager.PeerManager.peerAlert.Signal()
	manager.downloader.updatePiecePriority(peer)


}

func (manager *TorrentManager) HandlePieceMsg(msg PieceMsg) {
	//os.Exit(29)
	manager.downloader.putBlock(msg)

}

func (manager *TorrentManager) HandleRequestMsg(msg RequestMsg) {
}

func (manager *TorrentManager) HandleCancelMsg(msg CancelRequestMsg) {
}
