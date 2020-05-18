package main

import (
	"bufio"
	"encoding/binary"
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

	Pieces 			*Piece

	activeConnection map[string]*Peer
}

func NewTorrent(torrentPath string) *Torrent {

	var torrentFile = parser.Unmarshall(torrentPath)

	torrent := new(Torrent)
	torrent.activeConnection = make(map[string]*Peer, 0)
	torrent.TorrentFile = torrentFile
	torrent.TorrentFile.MapString["infoHash"] = parser.GetInfoHash(torrentFile)

	torrent.TrackerRequest(startedEvent)
	torrent.addPeers()

	fmt.Printf("%v\n", torrent.Peers[len(torrent.Peers)-1].ip)
	fmt.Printf("%v\n", torrent.Peers[len(torrent.Peers)-1].id)
	fmt.Printf("%v\n", torrent.Peers[len(torrent.Peers)-1].port)

	return torrent
}

func (torrent *Torrent) addPeers() {

	_, isPresent := torrent.TrackerResponse.MapList["peers"]

	if isPresent {
		for _, peer := range torrent.TrackerResponse.MapList["peers"].LDict {

			torrent.Peers = append(torrent.Peers, NewPeer(peer))

		}
	} else {

		peers := torrent.TrackerResponse.MapString["peers"]
		i := 0
		fmt.Printf("LEN %v \n", len(peers))
		for i < len(peers) {
			fmt.Printf("peers : %v\n", []byte(peers[i:(i+6)]))
			torrent.Peers = append(torrent.Peers, NewPeerFromString([]byte(peers[i:(i+6)])))
			i += 6
		}

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

func (torrent *Torrent) Listen() {
	sever, err := net.ListenTCP("tcp", utils.LocalAddr2)
	fmt.Printf("Listenning on %v\n",sever.Addr().String())

	for {
		if err == nil {
			connection, connErr := sever.AcceptTCP()

			println("newPeer\n")

			if connErr == nil {
				go torrent.handleNewPeer(connection)
			}

		} else {
			println("eer")
			log.Fatal(err)
		}

	}
}

func (torrent *Torrent) handleNewPeer(connection *net.TCPConn) {
	println("---------------------------------")
	_ = connection.SetKeepAlive(true)
	_ = connection.SetKeepAlivePeriod(utils.KeepAlliveDuration)

	mLen := 0
	handShakeMsg := make([]byte, 68)
	mLen, readFromConn := bufio.NewReader(connection).Read(handShakeMsg)
	fmt.Printf("read %v msg from %v : %v\n", mLen, connection.RemoteAddr(), handShakeMsg)

	_, handShakeErr := parser.ParseHandShake(handShakeMsg, torrent.TorrentFile.MapString["infoHash"])

	if handShakeErr != nil{
		fmt.Printf("%v \n closing connection \n",handShakeErr)
		_ = connection.Close()


	}


	n, writeErr := connection.Write(parser.GetMsg(parser.MSG{MsgID: parser.HandShakeMsg,InfoHash:[]byte(torrent.TorrentFile.MapString["infoHash"]),MyPeerID:utils.MyID}))

	fmt.Printf("wrote %v byte to %v with %v err", n,connection.RemoteAddr().String(),writeErr)
	i := 0
	for readFromConn == nil{

		incomingMsg := make([]byte,500)
		_, readFromConn = bufio.NewReader(connection).Read(incomingMsg)
	//	fmt.Printf("new msg # %v : %v\n", i, string(incomingMsg))

	//	fmt.Printf("new msg byte # %v : %v\n", i, incomingMsg)

		parsedMsg := parser.ParseMsg(incomingMsg)

		fmt.Printf("parsed msg \n msg ID %v LEN %v\n", parsedMsg.MsgID,parsedMsg.MsgLen)
		i++
	}

	println("---------------------------------")


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

func NewPeerFromString(peer []byte) *Peer {
	newPeer := new(Peer)
	newPeer.id = "N/A"

	ipByte := peer[0:4]

	ipString := ""
	for i := range ipByte {

		formatByte := make([]byte, 0)
		formatByte = append(formatByte, 0)
		formatByte = append(formatByte, ipByte[i])
		n := binary.BigEndian.Uint16(formatByte)
		fmt.Printf("num byte %d\n", formatByte)
		fmt.Printf("num %v\n", n)

		if i != len(ipByte)-1 {
			ipString += strconv.FormatUint(uint64(n), 10) + "."
		} else {
			ipString += strconv.FormatUint(uint64(n), 10)
		}
	}

	newPeer.ip = ipString

	n := binary.BigEndian.Uint16(peer[4:6])

	newPeer.port = strconv.FormatUint(uint64(n), 10)

	fmt.Printf("ip %v\n", newPeer.ip)
	fmt.Printf("port %v\n", newPeer.port)
	fmt.Printf("ip byte %#v\n", peer)

	return newPeer
}

func (peer *Peer) connectTo(torrent *Torrent) (*net.TCPConn, error) {

	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, err := net.DialTCP("tcp", utils.LocalAddr, remotePeerAddr)
	if err != nil {
		log.Fatal(err)
	}
	println("---------------------------------")
	fmt.Printf("Remote Addr %v \n", remotePeerAddr)

	msg := parser.MSG{MsgID: parser.HandShakeMsg, InfoHash: []byte(torrent.TorrentFile.MapString["infoHash"]), MyPeerID:utils.MyID}
	nByteWritten, writeErr := connection.Write(parser.GetMsg(msg))
	fmt.Printf("write %v bytes with %v err\n", nByteWritten, writeErr)

	handshakeBytes := make([]byte, 68)
	_, readFromConError := bufio.NewReader(connection).Read(handshakeBytes)

	fmt.Printf("Back HandShake %v\n", handshakeBytes)

	_, handShakeErr := parser.ParseHandShake(handshakeBytes, torrent.TorrentFile.MapString["infoHash"])

	if handShakeErr == nil && readFromConError == nil {
		torrent.activeConnection[remotePeerAddr.String()] = peer
	}else{
		log.Fatal("handshake Error")
		return nil,nil
	}

	println("---------------------------------")

	i := 0
	_, _ = connection.Write(parser.GetMsg(parser.MSG{MsgID: parser.UnchockeMsg}))

	for readFromConError == nil {
		incomingMsg := make([]byte, 500)

		_, readFromConError = bufio.NewReader(connection).Read(incomingMsg)

		parsedMsg := parser.ParseMsg(incomingMsg)

		fmt.Printf("parsed msg \n msg ID %v LEN %v\n", parsedMsg.MsgID,parsedMsg.MsgLen)



		i++
	}

	return connection, err
}


type Piece struct{

	infoHash string
	pieceLen int
	subPieceLen int
	subPiece [][]byte

}
