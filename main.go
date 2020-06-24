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
	"sync/atomic"

	//"DawnTorrent/ipc"
)

const (
	maxActiveTorrent = 3
)

const (
	initUi = 0
	updateUi = 0
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
	<-done
	println("non blocking")

}

func (dawnTorrentClient DawnTorrentClient) startZMQ(){
	context, _ := zmq.NewContext()

	server, _ := zmq.NewSocket(zmq.REP)

	errIpcServer := server.Bind("tcp://*:5555")

	if errIpcServer != nil{
		log.Fatal(errIpcServer)
	}
	fmt.Printf("\n context zmq %v\n", context)


	for {
		msgJSON, _ := server.Recv(zmq.SNDMORE)
		msg := new(IpcMsg)

		fmt.Printf("%v\n",string(msgJSON))
		errTorrentIPCData := json.Unmarshal([]byte(msgJSON),msg)

		if errTorrentIPCData != nil{
			log.Fatal(errTorrentIPCData)
		}

		dawnTorrentClient.CommandRouter(msg)
		res , _ := json.Marshal(msg)
		_, _ = server.Send(string(res), zmq.DONTWAIT)
		log.Println(msgJSON)

	}

}
func (dawnTorrentClient *DawnTorrentClient) addTorrent(torrentIpcData *PeerProtocol.TorrentIPCData){
	newTorrent := PeerProtocol.NewTorrent(torrentIpcData.Path, torrentIpcData.OpenMod)
	dawnTorrentClient.torrents["1"] = newTorrent
	dawnTorrentClient.CaptureDataForUI(newTorrent,torrentIpcData)
}
func (dawnTorrentClient DawnTorrentClient) CaptureDataForUI(torrent *PeerProtocol.Torrent,torrentIpcData *PeerProtocol.TorrentIPCData) {
	torrentIpcData.Len = torrent.File.FileLen
	torrentIpcData.Name = torrent.File.Name
	torrentIpcData.Status = int(atomic.LoadInt32(torrent.File.Status))
	torrentIpcData.CurrentLen = torrent.File.TotalDownloaded
	torrentIpcData.FileInfos =  torrent.File.Files
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
func (dawnTorrentClient *DawnTorrentClient) CommandRouter (msg *IpcMsg){
		println("------test------------")
	for _ , c := range msg.Commands {
		fmt.Printf("c command %v\n", c.Command)

		switch c.Command {
	case AddTorrent:
		 dawnTorrentClient.addTorrent(c.TorrentIPCData)
	case DeleteTorrent:
	case PauseTorrent:
		//return dawnTorrentClient.PauseTorrent(msg.infoHash)
	case ResumeTorrent:
		//return dawnTorrentClient.Resume(msg.infoHash)
	case InitClient:
		println("--------init data--------")
		 dawnTorrentClient.getClientInitData(msg)
	case CloseClient:

	}
	}
}


func initClient() *DawnTorrentClient {
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
				path := utils.GetPath(utils.TorrentDataPath,utils.GetFileName(f.Name()))
				torrent := PeerProtocol.NewTorrent(path,PeerProtocol.ResumeTorrentFile_)
				fmt.Printf("infohash %v\n",torrent.File.InfoHash)
				dawnTorrentClient.torrents[torrent.File.InfoHash] = torrent

				break
			}

		}


		fmt.Printf("len of init torrent %v",len(dawnTorrentClient.torrents))

	}
	go dawnTorrentClient.startZMQ()
	return dawnTorrentClient
}

func(dawnTorrentClient *DawnTorrentClient) getClientInitData(ipcMsg *IpcMsg ) *IpcMsg{
	ipcMsg.Commands = make([]*Command,0)
	println("getting init data")
	for _ , t := range dawnTorrentClient.torrents{

		command := new(Command)
		command.Command = InitClient
		command.TorrentIPCData = new(PeerProtocol.TorrentIPCData)

		dawnTorrentClient.CaptureDataForUI(t,command.TorrentIPCData)
		ipcMsg.Commands = append(ipcMsg.Commands,command)
	}

	return ipcMsg
}



type Command struct {
	Command        int
	TorrentIPCData *PeerProtocol.TorrentIPCData
}

type IpcMsg struct {
	Commands []*Command
}


