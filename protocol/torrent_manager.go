package protocol

import (
	"context"
	"log"
	"reflect"
)

const (
	StartTorrent    = iota
	StopTorrent     = iota
	CompleteTorrent = iota
)

type torrentManagerStateI interface {
	start()
	stop()
	complete()
}

type TorrentManager struct {
	torrent         *Torrent
	peerManager     *peerManager
	stopMsgPipeLine chan interface{}
	msgChan         chan BaseMsg
	torrentState    int
	uploaded        int
	totalDownloaded int
	left            int
	stateChan       chan int
	tracker         tracker
	state           int
	myState         torrentManagerStateI
}

func NewTorrentManager(torrentPath string) *TorrentManager {
	manager := new(TorrentManager)
	manager.torrent, _ = createNewTorrent(torrentPath)
	manager.msgChan = make(chan BaseMsg)
	manager.peerManager = newPeerManager(manager.msgChan, manager.torrent.InfoHashHex)
	manager.tracker = newTracker(manager.torrent.AnnouncerUrl, manager.torrent.InfoHashHex, manager.peerManager)

	manager.torrentState = StopTorrent
	manager.myState = &StoppedStated{manager: manager}

	manager.stateChan = make(chan int, 1)
	return manager
}

type startedStated struct {
	manager *TorrentManager
	cancelRoutines context.CancelFunc
}

func (state startedStated) start() {
	log.Printf("manager is already started")

}

func (state startedStated) stop() {
	state.cancelRoutines()
	state.manager.myState = &StoppedStated{manager: state.manager}
}

func (state startedStated) complete() {
	panic("implement me")
}

type StoppedStated struct {
	manager *TorrentManager
}

func (state StoppedStated) start() {

	ctx, cancelRoutine := context.WithCancel(context.TODO())

	go state.manager.msgRouter(ctx)
	go state.manager.peerManager.receiveOperation(ctx)
	go state.manager.tracker.starTracker(ctx)
	state.manager.peerManager.peerOperationReceiver <- startServer{
		swarm: state.manager.peerManager,
	}
	state.manager.myState = &startedStated{manager: state.manager,cancelRoutines: cancelRoutine}

	log.Printf("new state: %v", reflect.TypeOf(state.manager.myState))
}

func (state StoppedStated) stop() {
	log.Printf("manager is already stopped")
}

func (state StoppedStated) complete() {
	panic("implement me")
}

func (manager *TorrentManager) SetState(state int) {
	manager.stateChan <- state
}

func (manager *TorrentManager) Init() {

	for {
		manager.state = <-manager.stateChan
		log.Printf("new state : %v", manager.state)
		switch manager.state {
		case StartTorrent:
			manager.myState.start()
		case StopTorrent:
			manager.myState.stop()
		case CompleteTorrent:
			close(manager.stateChan)
		}
	}

}

func (manager TorrentManager) Stop() {
	close(manager.stateChan)
}

func (manager *TorrentManager) runPeriodicDownloader() {

}
func (manager *TorrentManager) msgRouter(ctx context.Context) {

	for {

		select {
		case <-ctx.Done():
			return
		case msg := <-manager.msgChan:
			msg.handleMsg(manager)
		}

	}

	/*switch msg.ID {
	case BitfieldMsg:
		// gotta check that bitfield is the correct len
		bitfieldCorrectLen := int(math.Ceil(float64(torrent.Downloader.nPiece) / 8.0))

		counter := 0
		if (len(msg.BitfieldRaw)) == bitfieldCorrectLen {

			pieceIndex := 0

			for i := 0; i < torrent.Downloader.nPiece; i += 8 {
				bitIndex := 7
				currentByte := msg.BitfieldRaw[i/8]
				for bitIndex >= 0 && pieceIndex < torrent.Downloader.nPiece {
					counter++
					currentBit := uint8(math.Exp2(float64(bitIndex)))
					bit := currentByte & currentBit

					isPieceAvailable := bit != 0
					torrent.peerManager.peerMutex.Lock()

					//TODO if a peer is removed, it is a problem if we try to access it
					// need to add verification that the peer is still in the map
					peer, isPresent := torrent.peerManager.PeersMap.Get(msg.Peer.id)
					if isPresent {
						peer.(*Peer).AvailablePieces[pieceIndex] = isPieceAvailable
					}
					torrent.peerManager.peerMutex.Unlock()

					// if piece available we put in the sorted map

					if isPieceAvailable {

						torrent.Downloader.pieceAvailabilityMutex.Lock()
						torrent.Downloader.Pieces[pieceIndex].Availability++
						torrent.Downloader.pieceAvailabilityMutex.Unlock()
					}
					pieceIndex++
					bitIndex--
				}

			}
		} else {
			//fmt.Printf("correctlen %v actual Len %v", bitfieldCorrectLen, len(msg.BitfieldRaw))
		}

		torrent.Downloader.SortPieceByAvailability()

	case InterestedMsg:
		torrent.peerManager.peerMutex.Lock()
		msg.Peer.interested = true
		torrent.peerManager.peerMutex.Unlock()

	case UnInterestedMsg:
		torrent.peerManager.peerMutex.Lock()
		msg.Peer.interested = false
		torrent.peerManager.peerMutex.Unlock()

	case UnchockeMsg:
		if msg.Length == unChokeMsgLen {
			torrent.peerManager.peerMutex.Lock()
			msg.Peer.peerIsChocking = false
			torrent.peerManager.peerMutex.Unlock()
		}

	case ChockedMsg:
		if msg.Length == chokeMsgLen {
			torrent.peerManager.peerMutex.Lock()
			msg.Peer.peerIsChocking = true
			torrent.chokeCounter++

			torrent.peerManager.peerMutex.Unlock()

		}
	case PieceMsg:
		// making sure that we are receiving a valid piece index
		if msg.PieceIndex < torrent.Downloader.nPiece {
			// verifies that the length of the data is not greater or smaller than amount requested
			if msg.PieceLen == torrent.Downloader.subPieceLength || msg.PieceLen == torrent.Downloader.Pieces[msg.PieceIndex].Len%torrent.Downloader.subPieceLength {
				//	_ = torrent.Downloader.AddSubPiece(msg, msg.Peer)
				torrent.Downloader.addPieceChannel <- msg
			}
		}

	case HaveMsg:
		if msg.Length == haveMsgLen && msg.PieceIndex < torrent.Downloader.nPiece {
			torrent.Downloader.pieceAvailabilityMutex.Lock()
			torrent.Downloader.Pieces[msg.PieceIndex].Availability++
			torrent.Downloader.pieceAvailabilityMutex.Unlock()
			torrent.Downloader.SortPieceByAvailability()

		}

	}
	*/
}

func (manager *TorrentManager) handleUnInterestedMsg(msg UnInterestedMsg) {
}

func (manager *TorrentManager) handleInterestedMsg(msg InterestedMsg) {
}

func (manager *TorrentManager) handleUnChokeMsg(msg UnChockedMsg) {
}
func (manager *TorrentManager) handleChokeMsg(msg ChockedMSg) {
}

func (manager *TorrentManager) handleHaveMsg(msg HaveMsg) {
}
func (manager *TorrentManager) handleBitFieldMsg(msg BitfieldMsg) {
}

func (manager *TorrentManager) handlePieceMsg(msg PieceMsg) {
}

func (manager *TorrentManager) handleRequestMsg(msg RequestMsg) {
}

func (manager *TorrentManager) handleCancelMsg(msg CancelRequestMsg) {
}
