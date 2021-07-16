package core

import (
	"DawnTorrent/core/subsService"
	"DawnTorrent/core/torrent"
	"DawnTorrent/core/tracker"
	"DawnTorrent/interfaces"
	"DawnTorrent/rpc/torrent_state"
	"context"
	log "github.com/sirupsen/logrus"
	"sync"
)

const (
	incrementAvailability = iota
	decrementAvailability = iota
)

type torrentManagerStateI interface {
	start()
	pause()
	stop()
	complete()
	getState() int
}

type TorrentManager struct {
	Id              string
	Torrent         *torrent.Torrent
	PeerManager     *PeerManager
	MsgChan         chan TorrentMsg
	Uploaded        int
	TotalDownloaded int
	Left            int
	stateChan       chan int
	scrapper        *tracker.Announcer
	state           int
	myState         torrentManagerStateI
	downloader      *Downloader
	subscribers     []subsService.Subscriber
}

func (manager *TorrentManager) GetSyncChan() chan interfaces.SyncOp {
	panic("implement me")
}

func (manager *TorrentManager) GetAnnounceList() []string {
	return manager.Torrent.AnnounceList
}

func (manager *TorrentManager) GetStats() (int, int, int) {
	return manager.TotalDownloaded, manager.Uploaded, manager.Left
}

func NewTorrentManager(torrentPath string) (*TorrentManager, error) {
	var err error
	manager := new(TorrentManager)
	manager.Torrent, err = torrent.CreateNewTorrent(torrentPath)

	if err != nil {
		return nil, err
	}
	manager.Id = manager.Torrent.InfoHashHex

	manager.MsgChan = make(chan TorrentMsg, 100)
	manager.stateChan = make(chan int, 1)
	manager.PeerManager = newPeerManager(manager.MsgChan, manager.Torrent.InfoHashHex, manager.Torrent.InfoHash)
	manager.downloader = NewTorrentDownloader(manager.Torrent, manager.PeerManager, manager.stateChan)
	newScrapper, err := tracker.NewAnnouncer(manager.Torrent.Announce, manager.Torrent.InfoHash, manager, manager.PeerManager)

	if err != nil {
		log.Fatal(err)
	}
	manager.scrapper = newScrapper

	manager.myState = &Stopped{manager: manager}

	return manager, nil
}

type InProgress struct {
	manager               *TorrentManager
	downloadCancelRoutine context.CancelFunc
	scrapperCancelFunc    context.CancelFunc
	msgRouterCancelFunc   context.CancelFunc
	downloadRoutineWait   *sync.WaitGroup
}

func (state *InProgress) getState() int {
	return interfaces.CompleteTorrent
}

func (state *InProgress) start() {
	log.Printf("manager is already started")

}

func (state *InProgress) stop() {
	state.downloadCancelRoutine()
	state.downloadRoutineWait.Wait()
	state.scrapperCancelFunc()
	state.msgRouterCancelFunc()

	state.manager.myState = &Stopped{manager: state.manager}
	log.Printf("Stopped....")
}

func (state *InProgress) pause() {
	state.downloadCancelRoutine()

	state.downloadRoutineWait.Wait()

	state.manager.myState = &Paused{manager: state.manager, downloadRoutineWait: &sync.WaitGroup{}, scrapperCancelFunc: state.scrapperCancelFunc, msgRouterCancelFunc: state.msgRouterCancelFunc}

	log.Printf("Paused.....")
}

func (state *InProgress) complete() {
	panic("implement me")
}

type Stopped struct {
	manager *TorrentManager
}

func (state *Stopped) getState() int {
	return interfaces.StopTorrent
}

func (state *Stopped) start() {

	downloaderCtx, cancelRoutine := context.WithCancel(context.TODO())
	scrapperCtx, scrapperCancelFunc := context.WithCancel(context.TODO())
	msgRouterCtx, msgRouterCancelFunc := context.WithCancel(context.TODO())
	downloadRoutineWait := &sync.WaitGroup{}
	downloadRoutineWait.Add(1)

	go state.manager.msgRouter(msgRouterCtx)
	go state.manager.scrapper.StartScrapper(scrapperCtx)
	go state.manager.downloader.Start(downloaderCtx, downloadRoutineWait)
	state.manager.myState = &InProgress{
		manager:               state.manager,
		downloadCancelRoutine: cancelRoutine,
		scrapperCancelFunc:    scrapperCancelFunc,
		msgRouterCancelFunc:   msgRouterCancelFunc,
		downloadRoutineWait:   downloadRoutineWait,
	}

	log.Printf("started...:")
}

func (state *Stopped) stop() {
	log.Printf("manager is already stopped")
}
func (state *Stopped) pause() {
	log.Printf("manager is already stopped")
}
func (state *Stopped) complete() {
	panic("implement me")
}

type Paused struct {
	manager               *TorrentManager
	downloadCancelRoutine context.CancelFunc
	scrapperCancelFunc    context.CancelFunc
	msgRouterCancelFunc   context.CancelFunc
	downloadRoutineWait   *sync.WaitGroup
}

func (state *Paused) start() {
	downloaderCtx, cancelRoutine := context.WithCancel(context.TODO())
	downloadRoutineWait := &sync.WaitGroup{}
	downloadRoutineWait.Add(1)
	go state.manager.downloader.Start(downloaderCtx, downloadRoutineWait)
	state.manager.myState = &InProgress{manager: state.manager, downloadCancelRoutine: cancelRoutine, downloadRoutineWait: downloadRoutineWait}
}

func (state *Paused) pause() {
	log.Printf("Already paused")
}

func (state *Paused) stop() {
	log.Panicf("cannot stop")
	state.scrapperCancelFunc()
	state.msgRouterCancelFunc()
	state.manager.myState = &Stopped{manager: state.manager}
}

func (state *Paused) complete() {
	panic("implement me")
}

func (state *Paused) getState() int {
	panic("implement me")
}

func (manager *TorrentManager) SetState(state int) {
	manager.stateChan <- state
}
func (manager *TorrentManager) GetState() int {
	return manager.myState.getState()
}

func (manager *TorrentManager) Init() {
	for {
		manager.state = <-manager.stateChan
		log.Printf("new subsService : %v", manager.state)
		switch manager.state {
		case interfaces.StartTorrent:
			manager.myState.start()
		case interfaces.PauseTorrent:
			manager.myState.pause()
		case interfaces.StopTorrent:
			manager.myState.stop()
		case interfaces.CompleteTorrent:
			manager.myState.stop()
			log.Printf("torrent is completed")

		}
		//manager.notify()
	}

}

func (manager TorrentManager) Stop() {
	close(manager.stateChan)
}

func (manager *TorrentManager) msgRouter(ctx context.Context) {

	for {

		select {
		case msg := <-manager.MsgChan:
			log.Debugf("routing msg..., msgID : %v", msg.getId())
			msg.handleMsg(manager)
		}

	}
}

func (manager *TorrentManager) serialize() *torrent_state.TorrentState {

	return &torrent_state.TorrentState{
		Infohash:  manager.Torrent.InfoHashHex,
		Torrent:   manager.Torrent.Serialize(),
		Stats:     manager.downloader.serialize(),
		Trackers:  manager.scrapper.Serialize(),
		PeerSwarm: manager.PeerManager.serialize(),
	}

}

func (manager *TorrentManager) notify() {
	for _, subscriber := range manager.subscribers {
		subscriber.UpdateState(manager.serialize())
	}
}

func (manager *TorrentManager) SubscribeToStateChange(subscriber subsService.Subscriber) {
	manager.subscribers = append(manager.subscribers, subscriber)
}
