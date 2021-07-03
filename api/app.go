package api

import (
	"DawnTorrent/api/torrent_state"
	"DawnTorrent/core"
	"DawnTorrent/interfaces"
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
	subscribers	 []torrent_state.ControlTorrent_SubscribeServer
}

func (a Control) createNewTorrent(torrentPath string) error {
	torrentManager, err := core.NewTorrentManager(torrentPath)
	if err != nil{
		return err
	}
	a.torrents[torrentManager.Id] = torrentManager

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
func (a Control) UpdateState(manager interfaces.TorrentManagerI) {

}

func (a Control) subScribeToTorrentState(subMsg *torrent_state.Subscription,stream torrent_state.ControlTorrent_SubscribeServer) {
	torrent, found  := a.torrents[subMsg.Infohash]

	if !found {
		return
	}
	a.subscribers = append(a.subscribers,stream)
	err := stream.Send(a.getTorrentState(torrent))
	if err != nil {
		log.Fatal(err)
		return
	}
}

func (a Control) getTorrentState(torrentManager *core.TorrentManager) *torrent_state.SingleTorrentState{
	return nil
}


