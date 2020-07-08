package main

import (
	"DawnTorrent/JobQueue"
	"DawnTorrent/PeerProtocol"
	"DawnTorrent/utils"
	"encoding/json"
	"fmt"
	zmq "github.com/pebbe/zmq4"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"sync/atomic"
	"time"

	//"DawnTorrent/ipc"
)

const (
	maxActiveTorrent = 3
)

const (
	clientCommand = 0
	torrentTorrent = 2
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

	/*
	wp := JobQueue.NewWorkerPool(15)

	wp.Start()


	go jobsMaker(wp)
	go jobsMaker(wp)


	*/
	<-done

}

func jobsMaker(worker *JobQueue.WorkerPool){
	newSeed := rand.NewSource(time.Now().UnixNano())
	random := rand.New(newSeed)
	_ = random.Intn(3)

	jobs  := 100
	for j := 0; j < jobs; j++{
		worker.AddJob(j)
		fmt.Printf("added job : %v\n", j)
		time.Sleep(time.Duration(time.Second.Seconds()*1000000000 * float64(1)))
	}
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

	//	fmt.Printf("%v\n",string(msgJSON))
		errTorrentIPCData := json.Unmarshal([]byte(msgJSON),msg)

		if errTorrentIPCData != nil{
			log.Fatal(errTorrentIPCData)
		}

		dawnTorrentClient.CommandRouter(msg)
		res , jsonErr := json.Marshal(msg)
		if jsonErr != nil{
			log.Fatal(jsonErr)
		}
		_, sendingError := server.Send(string(res), zmq.DONTWAIT)

		if sendingError != nil{
			log.Fatal(sendingError)
		}

	}

}
func (dawnTorrentClient *DawnTorrentClient) addTorrent(torrentIpcData *PeerProtocol.TorrentIPCData){
	newTorrent := PeerProtocol.NewTorrent(torrentIpcData.Path, torrentIpcData.AddMode)
	dawnTorrentClient.torrents[newTorrent.Downloader.InfoHashHex] = newTorrent
	dawnTorrentClient.CaptureDataForUI(newTorrent,torrentIpcData)
	go newTorrent.LifeCycle()
}
func (dawnTorrentClient DawnTorrentClient) CaptureDataForUI(torrent *PeerProtocol.Torrent,torrentIpcData *PeerProtocol.TorrentIPCData) {
	torrentIpcData.Len = torrent.Downloader.FileLength
	torrentIpcData.Name = torrent.Downloader.Name
	torrentIpcData.State = int(atomic.LoadInt32(torrent.Downloader.State))
	torrentIpcData.CurrentLen = torrent.Downloader.TotalDownloaded
	torrentIpcData.FileInfos =  torrent.Downloader.FileProperties
	torrentIpcData.InfoHash = torrent.Downloader.InfoHash
	torrentIpcData.InfoHashHex = torrent.Downloader.InfoHashHex
	println("my infoHash HEx")
	println(torrentIpcData.InfoHashHex)
	println(torrent.Downloader.InfoHash)
	torrentIpcData.DownloadRate = 	torrent.Downloader.DownloadRate

}
func (dawnTorrentClient *DawnTorrentClient) PauseTorrent (command *Command){
	fmt.Printf("\npause Test infoHash : %v\n", command.TorrentIPCData.InfoHashHex)
	fmt.Printf("torrent to pause \n %v\n",dawnTorrentClient.torrents[command.TorrentIPCData.InfoHashHex])
	torrent  := dawnTorrentClient.torrents[command.TorrentIPCData.InfoHashHex]
	fmt.Printf("torrent infoHash : %v\n",torrent.Downloader.InfoHashHex)
	torrent.StoppedState <- PeerProtocol.StoppedState
	command.TorrentIPCData.State = PeerProtocol.StoppedState
	fmt.Printf("pause TEST DONE")

}
func (dawnTorrentClient *DawnTorrentClient) Resume (command *Command){
	println("resuming........")
	dawnTorrentClient.torrents[command.TorrentIPCData.InfoHashHex].StartedState <- PeerProtocol.StartedState
	command.TorrentIPCData.State = PeerProtocol.StartedState
}

func (dawnTorrentClient *DawnTorrentClient) GetProgress (command *Command){
	fmt.Printf("\nInfoHashHex : %v\n",command.TorrentIPCData.InfoHashHex)

	command.TorrentIPCData.CurrentLen = dawnTorrentClient.torrents[command.TorrentIPCData.InfoHashHex].Downloader.TotalDownloaded
	command.TorrentIPCData.DownloadRate =  dawnTorrentClient.torrents[command.TorrentIPCData.InfoHashHex].Downloader.DownloadRate
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
				path := utils.GetPath(utils.TorrentDataPath,utils.GetFileName(f.Name()),f.Name())
				torrent := PeerProtocol.NewTorrent(path,PeerProtocol.ResumeTorrentFile_)
				fmt.Printf("\ninfohash %v\n",torrent.Downloader.InfoHashHex)
				torrent.Downloader.SetState(PeerProtocol.StoppedState)
				dawnTorrentClient.torrents[torrent.Downloader.InfoHashHex] = torrent
				fmt.Printf("torrent state %v, n neededPiece %v\n", torrent.Downloader.State, torrent.Downloader.Pieces[0].State)
				go torrent.LifeCycle()
				break
			}

		}


		fmt.Printf("len of init torrent %v",len(dawnTorrentClient.torrents))

	}
	go dawnTorrentClient.startZMQ()
	return dawnTorrentClient
}
func (dawnTorrentClient *DawnTorrentClient) CommandRouter (msg *IpcMsg){
	println("------test------------")

	if msg.CommandType >= 0{
		for _ , c := range msg.Commands {
			fmt.Printf("c command %v\n", c.Command)

			switch c.Command {
			case AddTorrent:
				dawnTorrentClient.addTorrent(c.TorrentIPCData)
			case DeleteTorrent:
			case PauseTorrent:
				dawnTorrentClient.PauseTorrent(c)
			case ResumeTorrent:
				 dawnTorrentClient.Resume(c)

			case GetProgress:
				dawnTorrentClient.GetProgress(c)
			case CloseClient:

			}
		}
	}else{
		switch msg.CommandType {
		case InitClient:
			println("getting init data--------------------")
			dawnTorrentClient.getClientInitData(msg)
		}
	}

}

func(dawnTorrentClient *DawnTorrentClient) getClientInitData(ipcMsg *IpcMsg) *IpcMsg{
	ipcMsg.Commands = make([]*Command,0)
	fmt.Printf("getting init data len %v\n", len(dawnTorrentClient.torrents))

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
	CommandType int
	Commands    []*Command
}


