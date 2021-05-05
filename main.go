package main

import (
	"DawnTorrent/protocol"


	//"DawnTorrent/ipc"
)

const (
	maxActiveTorrent = 3
)

const (
	clientCommand = 0
	torrentTorrent = 2
)



func main() {

	done := make(chan bool)


	//c.addTorrent("/home/jude/GolandProjects/DawnTorrent/files/ubuntu-20.04-desktop-amd64.iso.torrent",PeerProtocol.InitTorrentFile_)
	//c.torrents["1"].Start()

	torrentPath := "files/ubuntu-20.04-desktop-amd64.iso.torrent"
	protocol.NewTorrentManager(torrentPath)

	println("non blocking")

	/*
	wp := JobQueue.NewWorkerPool(15)

	wp.Start()


	go jobsMaker(wp)
	go jobsMaker(wp)


	*/
	<-done

}



