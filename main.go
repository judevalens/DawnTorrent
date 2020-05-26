package main

import (
	"torrent/PeerProtocol"
)

func main() {
	done := make(chan bool)

	torrentPath := "files/ubuntu-20.04-desktop-amd64.iso.torrent"
	torrent := PeerProtocol.NewTorrent(torrentPath,done)
	//go torrent.RequestQueueManager()
	_=torrent
	<-done
}
