package main

import (
	"DawnTorrent/app"
	"DawnTorrent/interfaces"
	"DawnTorrent/utils"
	"bufio"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"

	//"DawnTorrent/ipc"
)

const (
	maxActiveTorrent = 3
)

const (
	clientCommand = 0
	torrentTorrent = 2
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors: true,
		FullTimestamp: true,
	})
}

func main() {

	//done := make(chan bool)

	//c.addTorrent("/home/jude/GolandProjects/DawnTorrent/files/ubuntu-20.04-desktop-amd64.iso.torrent",PeerProtocol.InitTorrentFile_)
	//c.torrents["1"].Start()
	utils.InitDir()
	torrentPath := "files/cosmos-laundromat.torrent"
	 manager1 := app.NewTorrentManager(torrentPath)
	go manager1.Init()
	 manager1.SetState(interfaces.StartTorrent)
	println("non blocking")

	/*
	wp := JobQueue.NewWorkerPool(15)

	wp.Start()


	go jobsMaker(wp)
	go jobsMaker(wp)


	*/

	scanner := bufio.NewScanner(os.Stdin)
	var command string
	var i int = 0
	for command != "exit" {
		scanner.Scan()
		command = scanner.Text()
		fmt.Printf("%v: %v\n", i,command)
		state,err := strconv.Atoi(command)
			if err != nil{
				log.Printf("err: %v", err)
				continue
			}
			manager1.SetState(state)
		i++
	}

	manager1.Stop()

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

}





