package main

import (
	"DawnTorrent/api"
	"DawnTorrent/core"
	"DawnTorrent/core/torrent"
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
	clientCommand  = 0
	torrentTorrent = 2
)

func init() {
	//log.SetOutput(ioutil.Discard)

	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
}

func main() {

	go api.StartServer()
	//done := make(chan bool)

	//c.addTorrent("/home/jude/GolandProjects/DawnTorrent/files/ubuntu-20.04-desktop-amd64.iso.torrent",PeerProtocol.InitTorrentFile_)
	//c.torrents["1"].Start()
	utils.InitDir()
	torrentPath := "files/big-buck-bunny.torrent"

	newTorrent, err := torrent.CreateNewTorrent(torrentPath)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Infof("comment: %v\n", newTorrent.Comment)
	log.Infof("annouce list: %v\n", newTorrent.AnnounceList)
	log.Infof("info: %v\n", newTorrent.SingleInfo.Name)
	log.Infof("length : %v\n", newTorrent.SingleInfo.Length)
	log.Infof("piece length : %v\n", newTorrent.SingleInfo.PieceLength)
	log.Infof("mode : %v\n", newTorrent.FileMode)

	manager1, _ := core.NewTorrentManager(torrentPath)
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
		fmt.Printf("%v: %v\n", i, command)
		state, err := strconv.Atoi(command)
		if err != nil {
			log.Printf("err: %v", err)
			continue
		}

		_ = state
		//manager1.SetState(state)
		i++
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}

}
