package api

import (
	"DawnTorrent/core"
	"DawnTorrent/interfaces"
	"DawnTorrent/rpc/torrent_state"
	"log"
)

const (
	add         = iota
	remove      = iota
	create      = iota
	start       = iota
	stop        = iota
	subscribe   = iota
	unsubscribe = iota
)

type command func() (interface{},error)

func (c command) exec() (interface{},error) {
	return c()
}

type Control struct {
	torrents     map[string]*core.TorrentManager
	messageQueue []command
	subscribers	 []torrent_state.StateService_SubscribeServer
}

func (a Control)  UpdateState(state *torrent_state.TorrentState){
	for _, subscriber := range a.subscribers {
		err := subscriber.Send(state)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
	}
}

func (a Control) initClient() error {



	return nil
}
func (a Control) createNewTorrent(torrentPath string) error {
	torrentManager, err := core.NewTorrentManager(torrentPath)
	if err != nil{
		return err
	}
	a.torrents[torrentManager.Id] = torrentManager

	torrentManager.SubscribeToStateChange(a)

	return err
}
func (a Control) startTorrent(torrentId string) error{
	a.torrents[torrentId].SetState(interfaces.StartTorrent)
	return nil
}
func (a Control) stopTorrent(torrentId string) error {
	a.torrents[torrentId].SetState(interfaces.StopTorrent)
	return nil
}

func (a Control) subScribeToTorrentState(subMsg *torrent_state.Subscription,stream torrent_state.StateService_SubscribeServer) error {

	a.subscribers = append(a.subscribers,stream)

	return nil
}
func (a Control) getTorrentState(torrentManager *core.TorrentManager) *torrent_state.TorrentState{
	return nil
}


