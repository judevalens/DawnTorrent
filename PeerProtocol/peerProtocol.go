package PeerProtocol

import (
	"bufio"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"torrent/parser"
	"torrent/utils"
)

const (
	startedEvent   = "started"
	stoppedEvent   = "stopped"
	completedEvent = "completed"
	SubPieceLen    = 32000
	pieceHashLen   = 20
)

type Torrent struct {
	TrackerResponse *parser.Dict
	torrentMetaInfo *parser.Dict

	PeerSwarm		*PeerSwarm

	File *torrentFile

	activeConnection map[string]*Peer
}

func NewTorrent(torrentPath string) *Torrent {

	var torrentMetaInfo = parser.Unmarshall(torrentPath)

	torrent := new(Torrent)
	torrent.activeConnection = make(map[string]*Peer, 0)
	torrent.torrentMetaInfo = torrentMetaInfo
	torrent.torrentMetaInfo.MapString["infoHash"] = GetInfoHash(torrentMetaInfo)
	torrent.File = new(torrentFile)
	torrent.File.SubPieceLen = 1600
	//torrent.File.Pieces = redBlackTree.NewWithIntComparator()

	numPiece := int(math.Ceil(float64(len(torrent.torrentMetaInfo.MapDict["info"].MapString["pieces"]) / pieceHashLen)))

	torrent.File.SortedAvailability = make([]int,numPiece)

	for i := range torrent.File.SortedAvailability{
		torrent.File.SortedAvailability[i] = i
	}

	torrent.PeerSwarm = new(PeerSwarm)


	torrent.TrackerRequest(startedEvent)
	torrent.addPeers()

	fmt.Printf("%v\n", torrent.PeerSwarm.Peers[1].ip)
	fmt.Printf("%v\n", torrent.PeerSwarm.Peers[1].id)
	fmt.Printf("%v\n", torrent.PeerSwarm.Peers[1].port)

	randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
	randomPeer := randomSeed.Perm(len(torrent.PeerSwarm.Peers))
	maxPeer := math.Ceil((2.0 / 5.0) * float64(len(torrent.PeerSwarm.Peers)))
	fmt.Printf("max peer %v", maxPeer)

	for i := 0; i < int(maxPeer); i++ {
		go torrent.PeerSwarm.Peers[randomPeer[i]].ConnectTo(torrent)
	}

	return torrent
}

func GetInfoHash(dict *parser.Dict) string {

	// InnerStartingPosition leaves out the key

	startingPosition := dict.MapDict["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.MapDict["info"].KeyInfo.EndingPosition

	torrentFileString := parser.ToBencode(dict)

	torrentFileByte := []byte(torrentFileString)

	infoBytes := torrentFileByte[startingPosition:endingPosition]

	b := sha1.Sum(infoBytes)
	bSlice := b[:]

	println(hex.EncodeToString(bSlice))

	return string(bSlice)

}

func (torrent *Torrent) addPeers() {
	torrent.PeerSwarm.Peers = make([]*Peer,0)
	_, isPresent := torrent.TrackerResponse.MapList["peers"]
	if isPresent {
		for i, peer := range torrent.TrackerResponse.MapList["peers"].LDict {

			torrent.PeerSwarm.Peers = append(torrent.PeerSwarm.Peers, torrent.NewPeer(peer))
			torrent.PeerSwarm.SortedPeersIndex = append(torrent.PeerSwarm.SortedPeersIndex,i)
		}
	} else {

		peers := torrent.TrackerResponse.MapString["peers"]
		i := 0
		fmt.Printf("LEN %v \n", len(peers))
		for i < len(peers) {
			fmt.Printf("peers : %v\n", []byte(peers[i:(i+6)]))
			torrent.PeerSwarm.Peers = append(torrent.PeerSwarm.Peers, torrent.NewPeerFromString([]byte(peers[i:(i+6)])))
			torrent.PeerSwarm.SortedPeersIndex = append(torrent.PeerSwarm.SortedPeersIndex,i)

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
	infoHash := torrent.torrentMetaInfo.MapString["infoHash"]
	uploaded := 0
	downloaded := 0

	fmt.Printf("%x\n", torrent.torrentMetaInfo.MapString["infoHash"])

	left, _ := strconv.Atoi(torrent.torrentMetaInfo.MapDict["info"].MapString["length"])

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", infoHash)
	trackerRequestParam.Add("peer_id", PeerId)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(downloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	println("PARAM " + trackerRequestParam.Encode())

	trackerUrl := torrent.torrentMetaInfo.MapString["announce"] + "?" + trackerRequestParam.Encode()

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
	fmt.Printf("Listenning on %v\n", sever.Addr().String())

	for {
		if err == nil {
			_, connErr := sever.AcceptTCP()

			println("newPeer\n")

			if connErr == nil {
				//go torrent.handleNewPeer(connection)
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
	_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)

	mLen := 0
	handShakeMsg := make([]byte, 68)
	mLen, readFromConn := bufio.NewReader(connection).Read(handShakeMsg)
	if readFromConn != nil{
		_ = connection.Close()
	}
	fmt.Printf("read %v msg from %v : %v\n", mLen, connection.RemoteAddr(), handShakeMsg)

	_, handShakeErr := ParseHandShake(handShakeMsg, torrent.torrentMetaInfo.MapString["infoHash"])

	if handShakeErr != nil {
		fmt.Printf("%v \n closing connection \n", handShakeErr)
		_ = connection.Close()

	}

	n, writeErr := connection.Write(GetMsg(MSG{MsgID: HandShakeMsg, InfoHash: []byte(torrent.torrentMetaInfo.MapString["infoHash"]), MyPeerID: utils.MyID}))

	if writeErr != nil{
		_ = connection.Close()
	}

	fmt.Printf("wrote %v byte to %v with %v err", n, connection.RemoteAddr().String(), writeErr)
	i := 0
	for readFromConn == nil {

		incomingMsg := make([]byte, 500)
		_, readFromConn = bufio.NewReader(connection).Read(incomingMsg)
		//	fmt.Printf("new msg # %v : %v\n", i, string(incomingMsg))

		//	fmt.Printf("new msg byte # %v : %v\n", i, incomingMsg)

		parsedMsg := ParseMsg(incomingMsg, torrent)

		fmt.Printf("parsed msg \n msg ID %v LEN %v\n", parsedMsg.MsgID, parsedMsg.MsgLen)
		i++
	}

	println("---------------------------------")

}

func (torrent *Torrent) savePieces() {

	_, fileErr := os.OpenFile(utils.DawnTorrentHomeDir+"/"+torrent.torrentMetaInfo.MapDict["info"].MapString["name"], os.O_CREATE|os.O_RDWR, os.ModePerm)

	if fileErr != nil {
		log.Fatal(fileErr)
	}
}

func (torrent *Torrent) pieceSelection(msg MSG) {
	switch msg.MsgID {
	case BitfieldMsg:
		for i, b := range msg.BitfieldRaw {
			bitIndex := 7
			for bitIndex >= 0 {

				pieceIndex := (7 - bitIndex) + (i * 8)

				currentBit := uint8(math.Exp2(float64(bitIndex)))
				bit := b & currentBit

				if bit != 0 {
					torrent.File.Pieces[pieceIndex].Availability++
				}

				bitIndex--
			}

		}

	}
	for false {
		sort.Sort(torrent.PeerSwarm)
		numUnchockedPeer := 0
		i := 0
		for numUnchockedPeer < 3 && i < len(torrent.PeerSwarm.Peers){
			interestedPeer  := torrent.PeerSwarm.Peers[(len(torrent.PeerSwarm.SortedPeersIndex)-1-i)]
			for !interestedPeer.peerInterested && i < len(torrent.PeerSwarm.Peers){
				i++
				interestedPeer  = torrent.PeerSwarm.Peers[(len(torrent.PeerSwarm.SortedPeersIndex)-1-i)]
			}
			torrent.PeerSwarm.unChockedPeer[numUnchockedPeer].chocked = true
			torrent.PeerSwarm.unChockedPeer[numUnchockedPeer] = interestedPeer
			torrent.PeerSwarm.unChockedPeer[numUnchockedPeer].chocked = false
			i++
			numUnchockedPeer++
		}
		d, _ := time.ParseDuration("10s")
		time.Sleep(d)
	}

}

type PeerSwarm struct {
	Peers            []*Peer
	SortedPeersIndex []int
	unChockedPeer []*Peer
	interestedPeers  map[string]Peer
}

func(PeerSwarm *PeerSwarm)Len()int{
	return len(PeerSwarm.Peers)
}

func (PeerSwarm *PeerSwarm)Less(i,j int) bool {
	return PeerSwarm.Peers[PeerSwarm.SortedPeersIndex[i]].DownloadRate < PeerSwarm.Peers[PeerSwarm.SortedPeersIndex[j]].DownloadRate
}

func (PeerSwarm *PeerSwarm)Swap(i,j int){
	temp := PeerSwarm.SortedPeersIndex[i]
	PeerSwarm.SortedPeersIndex[i] =  PeerSwarm.SortedPeersIndex[j]
	PeerSwarm.SortedPeersIndex[j] = temp
}

type Peer struct {
	ip                string
	port              string
	id                string
	peerChocking      bool
	peerInterested    bool
	chocked           bool
	interested      	bool
	AvailablePieces   []bool
	numByteDownloaded int
	time              float64
	lastTimeStamp     time.Time
	DownloadRate      float64
	connection			*net.TCPConn
}

func (torrent *Torrent)NewPeer(dict *parser.Dict) *Peer {
	newPeer := new(Peer)
	newPeer.id = dict.MapString["peer id"]
	newPeer.port = dict.MapString["port"]
	newPeer.ip = dict.MapString["ip"]
	numPiece := int(math.Ceil(float64(len(torrent.torrentMetaInfo.MapDict["info"].MapString["pieces"]) / pieceHashLen)))

	newPeer.AvailablePieces = make([]bool,numPiece)

	return newPeer
}
func(torrent *Torrent) NewPeerFromString(peer []byte) *Peer {
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

func (peer *Peer) ConnectTo(torrent *Torrent) {

	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, err := net.DialTCP("tcp", nil, remotePeerAddr)
	if err != nil {
		log.Fatal(err)
	}
	println("---------------------------------------------")
	fmt.Printf("connected to peer at : %v from %v\n", remotePeerAddr.String(), connection.LocalAddr().String())

	msg := MSG{MsgID: HandShakeMsg, InfoHash: []byte(torrent.torrentMetaInfo.MapString["infoHash"]), MyPeerID: utils.MyID}
	nByteWritten, writeErr := connection.Write(GetMsg(msg))
	fmt.Printf("write %v bytes with %v err\n", nByteWritten, writeErr)

	handshakeBytes := make([]byte, 68)
	_, readFromConError := bufio.NewReader(connection).Read(handshakeBytes)

	fmt.Printf("Back HandShake %v\n", handshakeBytes)

	_, handShakeErr := ParseHandShake(handshakeBytes, torrent.torrentMetaInfo.MapString["infoHash"])

	if handShakeErr == nil && readFromConError == nil {
		torrent.activeConnection[remotePeerAddr.String()] = peer
	} else {
		fmt.Printf("%v\n", handShakeErr)
		_ = connection.Close()
	}

	i := 0
	_, _ = connection.Write(GetMsg(MSG{MsgID: UnchockeMsg}))

	for readFromConError == nil {
		incomingMsg := make([]byte, 500)

		_, readFromConError = bufio.NewReader(connection).Read(incomingMsg)

		parsedMsg := ParseMsg(incomingMsg, torrent)

		switch parsedMsg.MsgID {

		}

		fmt.Printf("msg# %v \nparsed msg \n msg ID %v LEN %v\n", i, parsedMsg.MsgID, parsedMsg.MsgLen)

		i++
		println("---------------------------------------------")

	}

}


type torrentFile struct {
	infoHash           string
	fileName           string
	fileLen            int
	currentPieceIndex  *Piece
	Pieces             []*Piece
	SortedAvailability []int
	nPieces            int
	SubPieceLen        int
}

func (torrentFile *torrentFile)Len()int{
	return len(torrentFile.SortedAvailability)
}

func (torrentFile *torrentFile)Less(i,j int)bool{
	return torrentFile.Pieces[i].Availability < torrentFile.Pieces[j].Availability
}

func (torrentFile *torrentFile)Swap(i, j int){
	temp := torrentFile.SortedAvailability[i]
	torrentFile.SortedAvailability[i]= torrentFile.SortedAvailability[j]
	torrentFile.SortedAvailability[j] = temp
}

func (torrent *Torrent) updatePiece(msg MSG) {
	peer := torrent.PeerSwarm.Peers[msg.PeerID]
	for i, b := range msg.BitfieldRaw {
		bitIndex := 7
		for bitIndex >= 0 {

			pieceIndex := (7 - bitIndex) + (i * 8)

			currentBit := uint8(math.Exp2(float64(bitIndex)))
			bit := b & currentBit
			msg.availablePiece[pieceIndex] = bit != 0
			peer.AvailablePieces[i] = bit != 0
			if bit != 0 {
				torrent.File.Pieces[pieceIndex].Availability++
			}

			bitIndex--
		}

	}
}

type Piece struct {
	PieceTotalLen uint32
	CurrentLen    uint32
	SubPieceLen   int
	Status        string
	SubPieces     [][]byte
	PieceIndex    uint32
	Availability  int
}

func (torrent *Torrent) addSubPiece(msg MSG, peer *Peer) error {
	var err error = nil
	subPieceIndex := int(math.Ceil(float64(msg.BeginIndex/SubPieceLen))) - 1

	if torrent.File.Pieces[msg.PieceIndex].Status != "empty" {
		torrent.File.Pieces[msg.PieceIndex].PieceTotalLen = msg.PieceLen
		if torrent.File.Pieces[msg.PieceIndex].PieceIndex != msg.PieceIndex {
			err = errors.New("wrong sub piece")
		}
	}

	if err != nil {
		return err
	}
	torrent.File.Pieces[msg.PieceIndex].CurrentLen += msg.PieceLen
	copy(torrent.File.Pieces[msg.PieceIndex].SubPieces[subPieceIndex], msg.Piece)

	if torrent.File.Pieces[msg.PieceIndex].CurrentLen == torrent.File.Pieces[msg.PieceIndex].PieceTotalLen{

		file, _ := os.OpenFile(utils.DawnTorrentHomeDir+"/"+torrent.torrentMetaInfo.MapDict["info"].MapString["name"], os.O_CREATE|os.O_RDWR, os.ModePerm)

		_, _ = file.WriteAt(msg.Piece, int64(msg.PieceIndex)*int64(torrent.File.SubPieceLen))
	}
	peer.numByteDownloaded = int(msg.PieceLen)
	peer.lastTimeStamp  = time.Now()
	peer.time += time.Now().Sub(peer.lastTimeStamp).Seconds()
	peer.DownloadRate = (float64(peer.numByteDownloaded)) / peer.time

	return err
}
