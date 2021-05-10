package app

import (
	"DawnTorrent/app/tracker"
	"DawnTorrent/protocol"
	"context"
	"log"
	"reflect"
	"sync"
)



type torrentManagerStateI interface {
	start()
	stop()
	complete()
	getState() int
}

type TorrentManager struct {
	Torrent         *Torrent
	PeerManager     *peerManager
	stopMsgPipeLine chan interface{}
	MsgChan         chan torrentMsg
	torrentState    int
	Uploaded        int
	TotalDownloaded int
	Left            int
	stateChan       chan int
	stateLock       *sync.Mutex
	scrapper        *tracker.Announcer
	state           int
	myState         torrentManagerStateI
}

func (manager *TorrentManager) GetAnnounceList() []string {
	return manager.Torrent.AnnounceList
}

func (manager *TorrentManager) GetStats() (int, int, int) {
	return manager.TotalDownloaded,manager.Uploaded,manager.Left
}


func NewTorrentManager(torrentPath string) *TorrentManager {
	manager := new(TorrentManager)
	manager.Torrent, _ = createNewTorrent(torrentPath)
	manager.MsgChan = make(chan torrentMsg)
	manager.PeerManager = newPeerManager(manager.MsgChan, manager.Torrent.InfoHashHex,manager.Torrent.infoHashByte[:])
	newScrapper, err := tracker.NewAnnouncer(manager.Torrent.AnnouncerUrl, manager.Torrent.InfoHash,manager,manager.PeerManager)

	if err != nil{
		log.Fatal(err)
	}

	manager.scrapper = newScrapper

	manager.torrentState = protocol.StopTorrent
	manager.myState = &StoppedStated{manager: manager}

	manager.stateChan = make(chan int, 1)
	return manager
}

type startedStated struct {
	manager *TorrentManager
	cancelRoutines context.CancelFunc
}

func (state startedStated) getState() int {
	return protocol.CompleteTorrent
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
	return protocol.StopTorrent
}

func (state StoppedStated) start() {

	ctx, cancelRoutine := context.WithCancel(context.TODO())

	go state.manager.msgRouter(ctx)
	go state.manager.PeerManager.receiveOperation(ctx)
	go state.manager.scrapper.StartScrapper(ctx)

	state.manager.myState = &startedStated{manager: state.manager,cancelRoutines: cancelRoutine}

	state.manager.PeerManager.PeerOperationReceiver <- startServer{
		swarm: state.manager.PeerManager,
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
func (manager *TorrentManager) GetState() int{
	return manager.myState.getState()
}

func (manager *TorrentManager) Init() {

	for {
		manager.state = <-manager.stateChan
		log.Printf("new state : %v", manager.state)
		switch manager.state {
		case protocol.StartTorrent:
			manager.myState.start()
		case protocol.StopTorrent:
			manager.myState.stop()
		case protocol.CompleteTorrent:
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
		case msg := <-manager.MsgChan:
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
	msg.getPeer().SetBitField(msg.Bitfield)

	for _,_ = range manager.Torrent.pieces{

		print("hahha\n")

	}
}

func (manager *TorrentManager) handlePieceMsg(msg PieceMsg) {
}

func (manager *TorrentManager) handleRequestMsg(msg RequestMsg) {
}

func (manager *TorrentManager) handleCancelMsg(msg CancelRequestMsg) {
}