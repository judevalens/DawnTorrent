package main

import (
	"DawnTorrent/PeerProtocol"
	"DawnTorrent/ipc"

	//"DawnTorrent/ipc"
)

const (
	maxActiveTorrent = 3
)

type client struct {

	torrents []*PeerProtocol.Torrent

}

func main() {
//	done := make(chan bool)

		ipc.InitZMQ()
	///torrentPath := "/home/jude/GolandProjects/DawnTorrent/files/ubuntu-20.04-desktop-amd64.iso.torrent"
	//torrent := PeerProtocol.NewTorrent(torrentPath, done)

	//go DawnTorrent.RequestQueueManager()
	//_ = torrent
	//<-done
}
