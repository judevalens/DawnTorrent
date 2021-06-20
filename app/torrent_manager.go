package app

import (
	"DawnTorrent/app/tracker"
	"DawnTorrent/interfaces"
	"context"
	"log"
	"reflect"
	"sync"
)

const (
	incrementAvailability = iota
	decrementAvailability = iota
)

type torrentManagerStateI interface {
	start()
	stop()
	complete()
	getState() int
}

type TorrentManager struct {
	Torrent         *Torrent
	PeerManager     *PeerManager
	stopMsgPipeLine chan interface{}
	MsgChan         chan TorrentMsg
	torrentState    int
	Uploaded        int
	TotalDownloaded int
	Left            int
	stateChan       chan int
	stateLock       *sync.Mutex
	scrapper        *tracker.Announcer
	state           int
	myState         torrentManagerStateI
	downloader      *Downloader
	SyncOperation   chan interfaces.SyncOp

}

func (manager *TorrentManager) GetSyncChan() chan interfaces.SyncOp {
	panic("implement me")
}

func (manager *TorrentManager) GetAnnounceList() []string {
	return manager.Torrent.AnnounceList
}

func (manager *TorrentManager) GetStats() (int, int, int) {
	return manager.TotalDownloaded,manager.Uploaded,manager.Left
}

func NewTorrentManager(torrentPath string) *TorrentManager {
	manager := new(TorrentManager)
	manager.Torrent, _ = CreateNewTorrent(torrentPath)
	manager.MsgChan = make(chan TorrentMsg)
	manager.SyncOperation = make(chan interfaces.SyncOp)
	manager.PeerManager = newPeerManager(manager.MsgChan, manager.Torrent.InfoHashHex,manager.Torrent.InfoHashByte[:])
	manager.downloader = NewTorrentDownloader(manager.Torrent,manager,manager.PeerManager)
	newScrapper, err := tracker.NewAnnouncer(manager.Torrent.AnnouncerUrl, manager.Torrent.InfoHash,manager,manager.PeerManager)

	if err != nil{
		log.Fatal(err)
	}

	manager.scrapper = newScrapper

	manager.torrentState = interfaces.StopTorrent
	manager.myState = &StoppedStated{manager: manager}

	manager.stateChan = make(chan int, 1)

	return manager
}

type startedStated struct {
	manager *TorrentManager
	cancelRoutines context.CancelFunc
}

func (state startedStated) getState() int {
	return interfaces.CompleteTorrent
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
	return interfaces.StopTorrent
}

func (state StoppedStated) start() {

	ctx, cancelRoutine := context.WithCancel(context.TODO())

	go state.manager.msgRouter(ctx)
	go state.manager.receiveOperation(ctx)
	go state.manager.PeerManager.receiveOperation(ctx)
	go state.manager.scrapper.StartScrapper(ctx)
	go state.manager.downloader.Start(ctx)
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
		case interfaces.StartTorrent:
			manager.myState.start()
		case interfaces.StopTorrent:
			manager.myState.stop()
		case interfaces.CompleteTorrent:
			close(manager.stateChan)
		}
	}

}

func (manager TorrentManager) Stop() {
	close(manager.stateChan)
}

func (manager *TorrentManager) msgRouter(ctx context.Context) {

	for {

		select {
		case <-ctx.Done():
			return
		case msg := <-manager.MsgChan:
			log.Printf("routing msg..., msgID : %v", msg.getId())
			msg.handleMsg(manager)
		}

	}
}

func (manager *TorrentManager) receiveOperation(ctx context.Context) {

	for {

		select {
		case <-ctx.Done():
			return
		case operation := <-manager.SyncOperation:
			operation()
		}

	}
}