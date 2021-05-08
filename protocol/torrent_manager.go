package protocol

import (
	"context"
	"log"
	"reflect"
	"sync"
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
	getState() int
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
	stateLock 		*sync.Mutex
	scrapper         *Scrapper
	state           int
	myState         torrentManagerStateI
}

func NewTorrentManager(torrentPath string) *TorrentManager {
	manager := new(TorrentManager)
	manager.torrent, _ = createNewTorrent(torrentPath)
	manager.msgChan = make(chan BaseMsg)
	manager.peerManager = newPeerManager(manager.msgChan, manager.torrent.InfoHashHex,manager.torrent.infoHashByte[:])
	newScrapper, err := newTracker(manager.torrent.AnnouncerUrl, manager.torrent.InfoHash,manager)

	if err != nil{
		log.Fatal(err)
	}

	manager.scrapper = newScrapper

	manager.torrentState = StopTorrent
	manager.myState = &StoppedStated{manager: manager}

	manager.stateChan = make(chan int, 1)
	return manager
}

type startedStated struct {
	manager *TorrentManager
	cancelRoutines context.CancelFunc
}

func (state startedStated) getState() int {
	return CompleteTorrent
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

func (state StoppedStated) getState() int {
	return StopTorrent
}

func (state StoppedStated) start() {

	ctx, cancelRoutine := context.WithCancel(context.TODO())

	go state.manager.msgRouter(ctx)
	go state.manager.peerManager.receiveOperation(ctx)
	go state.manager.scrapper.startScrapper(ctx)

	state.manager.myState = &startedStated{manager: state.manager,cancelRoutines: cancelRoutine}

	state.manager.peerManager.peerOperationReceiver <- startServer{
		swarm: state.manager.peerManager,
	}

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
func (manager *TorrentManager) getState() int{
	return manager.myState.getState()
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
			log.Printf("routing msg...: %v", msg)
			msg.handleMsg(manager)
		}

	}
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