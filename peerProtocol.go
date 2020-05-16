package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"torrent/parser"
	"torrent/utils"
)

const (
	startedEvent   = "started"
	stoppedEvent   = "stopped"
	completedEvent = "completed"
)



type Torrent struct {
	TrackerResponse *parser.Dict
	TorrentFile     *parser.Dict
	Peers           []*Peer

	incomingCon 	map[string]*Peer
	outgoingCon		map[string]*Peer


}

func NewTorrent(torrentPath string) *Torrent {

	var torrentFile = parser.Unmarshall(torrentPath)

	torrent := new(Torrent)
	torrent.TorrentFile = torrentFile
	torrent.TorrentFile.MapString["infoHash"] = parser.GetInfoHash(torrentFile)

	torrent.TrackerRequest(startedEvent)

	fmt.Printf("%v\n", torrent.TrackerResponse.MapList["peers"].LDict[0].MapString["ip"])
	fmt.Printf("%v\n", torrent.TrackerResponse.MapList["peers"].LDict[0].MapString["peer id"])
	fmt.Printf("%v\n", torrent.TrackerResponse.MapList["peers"].LDict[0].MapString["port"])

	return torrent
}

func (torrent *Torrent) addPeers() {
	for _, peer := range torrent.TrackerResponse.MapList["peers"].LDict {

		torrent.Peers = append(torrent.Peers, NewPeer(peer))

	}

}

func openTorrent(torrentPath string) *Torrent {
	return new(Torrent)
}

func (torrent *Torrent) saveTorrent() {
}

func (torrent *Torrent) TrackerRequest(event string) {

	PeerId := utils.MyID
	infoHash := torrent.TorrentFile.MapString["infoHash"]
	uploaded := 0
	downloaded := 0

	fmt.Printf("%x\n", torrent.TorrentFile.MapString["infoHash"])

	left, _ := strconv.Atoi(torrent.TorrentFile.MapDict["info"].MapString["length"])

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", infoHash)
	trackerRequestParam.Add("peer_id", PeerId)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(downloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	println("PARAM " + trackerRequestParam.Encode())

	trackerUrl := torrent.TorrentFile.MapString["announce"] + "?" + trackerRequestParam.Encode()

	trackerResponseByte, _ := http.Get(trackerUrl)
	trackerResponse, _ := ioutil.ReadAll(trackerResponseByte.Body)

	fmt.Printf("%v\n", string(trackerResponse))

	trackerDictResponse := parser.UnmarshallFromArray(trackerResponse)

	_, isPresent := trackerDictResponse.MapString["failure reason"]

	if isPresent {
		err := errors.New(trackerDictResponse.MapString["failure reason"])

		log.Fatal(err)

	} else {
		torrent.TrackerResponse = trackerDictResponse
	}
}

type Peer struct {
	ip   string
	port string
	id   string
}

func NewPeer(dict *parser.Dict) *Peer {
	newPeer := new(Peer)
	newPeer.id = dict.MapString["peer id"]
	newPeer.port = dict.MapString["port"]
	newPeer.ip = dict.MapString["ip"]
	return newPeer
}

func (peer *Peer) connectTo() (*net.TCPConn, error) {
	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, err := net.DialTCP("tcp", utils.LocalAddr, remotePeerAddr)
		if err != nil{
			log.Fatal(err)
		}
	fmt.Printf("addr %v \n",remotePeerAddr)
	fmt.Printf("con %v \n",connection)

	return connection, err
}


func Listen() {
	sever, err := net.Listen("tcp", ":"+strconv.Itoa(utils.PORT))
	println("eer")

	for {
		if err == nil {
			connection, connErr := sever.Accept()

			println("newPeer\n")

			if connErr == nil {
				go handleNewPeer(connection)
			}

		} else {
			println("eer")
			log.Fatal(err)
		}

	}
}
func handleNewPeer(conn net.Conn) {

	mLen := 0
		b := make([]byte, 150)
		mLen, _ = bufio.NewReader(conn).Read(b)
		fmt.Printf("%v\n", conn.RemoteAddr().String())
		fmt.Printf("read %v msg from : %v\n", mLen, b)

}