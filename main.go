package main

import (
	"sync"
	"torrent/PeerProtocol"
)

func main() {

	waiting := new(sync.WaitGroup)
	torrentPath := "files/ubuntu-20.04-live-server-amd64.iso.torrent"
	waiting.Add(1)
	torrent := PeerProtocol.NewTorrent(torrentPath)
	torrent.Listen()
	waiting.Wait()
}
