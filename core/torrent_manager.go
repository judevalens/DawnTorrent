package core

import (
	"DawnTorrent/core/state"
	"DawnTorrent/core/torrent"
	"DawnTorrent/core/tracker"
	"DawnTorrent/interfaces"
	"context"
	log "github.com/sirupsen/logrus"
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
	subscribers 	[]state.Subscriber
}

func (manager *TorrentManager) notifySubs(){
	for _, subscriber := range manager.subscribers {
		subscriber.UpdateState(manager)
	}
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

	manager.myState = &StoppedStated{manager: manager}

	return manager, nil
}

type startedStated struct {
	manager               *TorrentManager
	cancelDownloadRoutine context.CancelFunc
	wait                  *sync.WaitGroup
}

func (state startedStated) getState() int {
	return interfaces.CompleteTorrent
}

func (state startedStated) start() {
	log.Printf("manager is already started")

}

func (state startedStated) stop() {
	state.cancelDownloadRoutine()

	state.wait.Wait()

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

	downloaderCtx, cancelRoutine := context.WithCancel(context.TODO())
	scrapperCtx, _ := context.WithCancel(context.TODO())
	msgRouterCtx, _ := context.WithCancel(context.TODO())
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)

	go state.manager.msgRouter(msgRouterCtx)
	go state.manager.scrapper.StartScrapper(scrapperCtx)
	go state.manager.downloader.Start(downloaderCtx, waitGroup)
	state.manager.myState = &startedStated{manager: state.manager, cancelDownloadRoutine: cancelRoutine, wait: waitGroup}

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
func (manager *TorrentManager) GetState() int {
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
			manager.myState.stop()
			log.Printf("torrent is completed")

		}
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
