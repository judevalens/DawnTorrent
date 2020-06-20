package main

import (
	"DawnTorrent/PeerProtocol"
	"DawnTorrent/utils"
	"encoding/json"
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"io/ioutil"
	"log"
	"path/filepath"

	//"DawnTorrent/ipc"
)

const (
	maxActiveTorrent = 3
)

type DawnTorrentClient struct {

	torrents map[string]*PeerProtocol.Torrent

}

func main() {
	done := make(chan bool)
	 initClient()

	//c.addTorrent("/home/jude/GolandProjects/DawnTorrent/files/ubuntu-20.04-desktop-amd64.iso.torrent",PeerProtocol.InitTorrentFile_)
	//c.torrents["1"].Start()


//


	println("non blocking")
	<-done
}

func (dawnTorrentClient DawnTorrentClient) InitZMQ(){
	context, _ := zmq.NewContext()

	server, _ := zmq.NewSocket(zmq.REP)

	errIpcServer := server.Bind("tcp://*:5555")

	if errIpcServer != nil{
		log.Fatal(errIpcServer)
	}
	fmt.Printf("\n context zmq %v\n", context)


	for {
		msgJSON, _ := server.Recv(zmq.SNDMORE)

		msg := new(TorrentIPCData)
		errTorrentIPCData := json.Unmarshal([]byte(msgJSON),msg)

		if errTorrentIPCData != nil{
			log.Fatal(errTorrentIPCData)
		}
		dawnTorrentClient.CommandRouter(msg)

		_, _ = server.Send("msg "+msgJSON, zmq.DONTWAIT)
		log.Println(msg)

	}

}
func (dawnTorrentClient *DawnTorrentClient) addTorrent(path string,mod int){
	newTorrent := PeerProtocol.NewTorrent(path, mod)
	dawnTorrentClient.torrents["1"] = newTorrent
}
func (dawnTorrentClient *DawnTorrentClient) PauseTorrent (hash string){
	dawnTorrentClient.torrents[hash].Pause()
}
func (dawnTorrentClient *DawnTorrentClient) Resume (hash string){
	dawnTorrentClient.torrents[hash].ResumeFromPause()
}
func (dawnTorrentClient *DawnTorrentClient) Start(hash string){
	dawnTorrentClient.torrents[hash].Start()
}
func (dawnTorrentClient *DawnTorrentClient) CommandRouter (msg *TorrentIPCData){
	switch msg.command {
	case AddTorrent:
		dawnTorrentClient.addTorrent(msg.path,PeerProtocol.InitTorrentFile_)
	case DeleteTorrent:
	case PauseTorrent:
		dawnTorrentClient.PauseTorrent(msg.infohash)
	case ResumeTorrent:
		dawnTorrentClient.Resume(msg.infohash)
	case InitClient:
		initClient()
	case CloseClient:

	}
}


func initClient() *DawnTorrentClient{
	dawnTorrentClient := new(DawnTorrentClient)
	dawnTorrentClient.torrents = make(map[string]*PeerProtocol.Torrent)
	file, err := ioutil.ReadDir(utils.SavedTorrentDir)
	fmt.Printf("file %v err %v\n",file,err )

	for _ , filesss := range file {
		fmt.Printf("parenttt %v \n",filesss.Name() )

		savedTorrent, _:= ioutil.ReadDir(filepath.FromSlash(utils.SavedTorrentDir+ "/" + filesss.Name()))


		for _ , f := range  savedTorrent {
			fmt.Printf("child %v \n",f.Name())
			if filepath.Ext(f.Name()) == ".json" {
				extensionLen := len(filepath.Ext(f.Name()))
				fileNameLen := len(f.Name())
				torrent := PeerProtocol.NewTorrent(f.Name()[:fileNameLen-extensionLen],PeerProtocol.ResumeTorrentFile_)
				fmt.Printf("infohash %v\n",torrent.File.InfoHash)
				dawnTorrentClient.torrents[torrent.File.InfoHash] = torrent
				break
			}

		}

	}
	//go dawnTorrentClient.InitZMQ()
	return dawnTorrentClient
}

type TorrentIPCData struct {
	command int
	name string
	path string
	infohash string
	len string
	currentLen string
	piecesStatus []bool
}


