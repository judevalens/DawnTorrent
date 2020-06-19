package main

import (
	"DawnTorrent/PeerProtocol"
	"DawnTorrent/ipc"
)

func main() {
	done := make(chan bool)

	ipc.InitZeroMQ()
	torrentPath := "C:\\Users\\jude\\go\\src\\DawnTorrent\\files\\big-buck-bunny.torrent"
	torrent := PeerProtocol.NewTorrent(torrentPath, done)
	//go DawnTorrent.RequestQueueManager()
	_ = torrent
	<-done
}