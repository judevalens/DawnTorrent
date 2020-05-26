package PeerProtocol

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"torrent/parser"
	"torrent/utils"
)

type PeerSwarm struct {
	Peers               []*Peer
	PeersMap            map[string]*Peer
	SortedPeersIndex    []int
	interestedPeerIndex []int
	interestedPeerMap   map[string]*Peer
	interestingPeer     map[string]*Peer
	unChockedPeer       []*Peer
	unChockedPeerMap    map[string]*Peer
	peerMutex           sync.Mutex
	interestingPeerMutex           sync.Mutex
	activeConnection  map[string]*Peer
	maxConnection  int32
	nActiveConnection *int32

	torrent    *Torrent
}

func NewPeerSwarm(torrent *Torrent) *PeerSwarm {
	newPeerSwarm := new(PeerSwarm)
	newPeerSwarm.PeersMap = make(map[string]*Peer)
	newPeerSwarm.interestingPeer = make(map[string]*Peer)
	newPeerSwarm.interestedPeerMap = make(map[string]*Peer)
	newPeerSwarm.unChockedPeer = make([]*Peer, 4)
	//newPeerSwarm.unChockedPeerMap = make(map[string]*Peer,0)
	newPeerSwarm.activeConnection = make(map[string]*Peer, 0)
	newPeerSwarm.maxConnection = 70
	newPeerSwarm.nActiveConnection = new(int32)
	atomic.AddInt32(newPeerSwarm.nActiveConnection,0)
	newPeerSwarm.torrent = torrent

	newPeerSwarm.initConnection()
	return newPeerSwarm
}

func (peerSwarm *PeerSwarm) Listen() {
	sever, err := net.ListenTCP("tcp", utils.LocalAddr2)
	fmt.Printf("Listenning on %v\n", sever.Addr().String())

	for {
		if err == nil {
			connection, connErr := sever.AcceptTCP()
			if connErr == nil {
				// limits the number of active connection
				if  *peerSwarm.nActiveConnection <= peerSwarm.maxConnection{
					println("newPeer\n")

					go peerSwarm.handleNewPeer(connection)
				}else{
					println(peerSwarm.nActiveConnection)
					println(peerSwarm.maxConnection)
					println("max connection reached!!!!!")
				}
			} else {
				_ = connection.Close()
			}
		} else {
			println("eer")
			log.Fatal(err)
		}

	}
}

func (peerSwarm *PeerSwarm) handleNewPeer(connection *net.TCPConn) {
	println("---------------------------------")
	_ = connection.SetKeepAlive(true)
	_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)
	var newPeer *Peer

	mLen := 0
	handShakeMsg := make([]byte, 68)
	mLen, readFromConn := bufio.NewReader(connection).Read(handShakeMsg)
	if readFromConn == nil {
		fmt.Printf("read %v msg from %v : %v\n", mLen, connection.RemoteAddr(), handShakeMsg)

		parsedHandShakeMsg, handShakeErr := ParseHandShake(handShakeMsg, peerSwarm.torrent.File.infoHash)

		if handShakeErr == nil {
			remotePeerAddr, _ := net.ResolveTCPAddr("tcp", connection.RemoteAddr().String())
			fmt.Printf("incoming connection from %v\n", remotePeerAddr.String())
			//when we receive a connection request , we creat initialize a new peer struct
			newPeer = peerSwarm.addNewPeer(remotePeerAddr.IP.String(),strconv.Itoa(remotePeerAddr.Port),parsedHandShakeMsg.peerID)
			newPeer.connection = connection
			peerSwarm.activeConnection[newPeer.id] = newPeer
			n, writeErr := connection.Write(GetMsg(MSG{MsgID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.File.infoHash), MyPeerID: utils.MyID}))
			fmt.Printf("wrote %v byte to %v with %v err", n, connection.RemoteAddr().String(), writeErr)
			defer peerSwarm.DropConnection(newPeer)
			if writeErr == nil {
				i := 0
				for readFromConn == nil {

					incomingMsg := make([]byte, 32000)
					_, readFromConn = bufio.NewReader(connection).Read(incomingMsg)
					//	fmt.Printf("new msg # %v : %v\n", i, string(incomingMsg))

					//	fmt.Printf("new msg byte # %v : %v\n", i, incomingMsg)

					parsedMsg , parserMsgErr := ParseMsg(incomingMsg, newPeer)


					//TODO
					// if peer send a wrong packet, we close the connection, I know it's a little bit extreme; we shall fix that later
					if parserMsgErr != nil{
						_ = connection.Close()
						break
					}

					peerSwarm.torrent.msgRouter(parsedMsg)

					fmt.Printf("parsed msg \n msg ID %v LEN %v\n", parsedMsg.MsgID, parsedMsg.MsgLen)
					i++

					println("---------------------------------")

				}
			}


		} else {

			fmt.Printf("%v \n closing connection \n", handShakeErr)
			_ = connection.Close()
		}

	} else {
		_ = connection.Close()
	}

}



func (peerSwarm *PeerSwarm) addPeersFromTracker(peers *parser.Dict) {
	peerSwarm.Peers = make([]*Peer, 0)
	_, isPresent := peers.MapList["peers"]
	if isPresent {
		for _, peer := range peers.MapList["peers"].LDict {

			peerSwarm.addNewPeer(peer.MapString["ip"],peer.MapString["port"],peer.MapString["peer id"])
		}
	} else {

		peers := peers.MapString["peers"]
		i := 0
		fmt.Printf("LEN %v \n", len(peers))
		for i < len(peers) {
			fmt.Printf("peers : %v\n", []byte(peers[i:(i+6)]))

			ip,port,id := peerSwarm.NewPeerFromString([]byte(peers[i:(i + 6)]))
			peerSwarm.addNewPeer(ip,port,id)

			i += 6
		}

	}

}

func (peerSwarm *PeerSwarm) TrackerRequest(event string)(*parser.Dict,error) {
	var err error
	PeerId := utils.MyID
	infoHash := peerSwarm.torrent.File.infoHash
	uploaded := 0
	downloaded := 0

	fmt.Printf("%x\n", infoHash)

	left :=peerSwarm.torrent.File.fileLen

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", infoHash)
	trackerRequestParam.Add("peer_id", PeerId)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(downloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	println("PARAM " + trackerRequestParam.Encode())

	trackerUrl := peerSwarm.torrent.File.torrentMetaInfo.MapString["announce"] + "?" + trackerRequestParam.Encode()

	trackerResponseByte, _ := http.Get(trackerUrl)
	trackerResponse, _ := ioutil.ReadAll(trackerResponseByte.Body)

	fmt.Printf("%v\n", string(trackerResponse))

	trackerDictResponse := parser.UnmarshallFromArray(trackerResponse)

	_, isPresent := trackerDictResponse.MapString["failure reason"]

	if isPresent {
		err = errors.New(trackerDictResponse.MapString["failure reason"])

	}

	return trackerDictResponse,err

}


func (peerSwarm *PeerSwarm)initConnection(){

	peersFromTracker , err := peerSwarm.TrackerRequest(startedEvent)

	if err == nil{
		peerSwarm.addPeersFromTracker(peersFromTracker)
		randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomPeer := randomSeed.Perm(len(peerSwarm.Peers))
		maxPeer := math.Ceil((5 / 5.0) * float64(len(peerSwarm.Peers)))
		fmt.Printf("max peer %v", maxPeer)

		for i := 0; i < int(maxPeer); i++ {
			go peerSwarm.Peers[randomPeer[i]].ConnectTo(peerSwarm)
		}
	}

	go peerSwarm.Listen()

}
func (peerSwarm *PeerSwarm) NewPeer(dict *parser.Dict) *Peer {
	newPeer := new(Peer)
	newPeer.id = dict.MapString["peer id"]
	newPeer.port = dict.MapString["port"]
	newPeer.ip = dict.MapString["ip"]
	newPeer.peerChocking = false
	newPeer.interested = false
	newPeer.AvailablePieces = make([]bool, int(math.Ceil(float64(peerSwarm.torrent.File.nPiece)/8.0)*8))

	return newPeer
}
func (peerSwarm *PeerSwarm) NewPeerFromString(peer []byte) (string,string,string) {
	newPeer := new(Peer)

	newPeer.AvailablePieces = make([]bool, peerSwarm.torrent.File.nPiece)

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

	portBytes := binary.BigEndian.Uint16(peer[4:6])

	return ipString,strconv.FormatUint(uint64(portBytes), 10),ipString + ":" + newPeer.port
}
func (peerSwarm *PeerSwarm) DropConnection(peer *Peer) {
	peerSwarm.peerMutex.Lock()
	delete(peerSwarm.PeersMap,peer.id)
	peerSwarm.activeConnection[peer.connection.RemoteAddr().String()] = nil
	atomic.AddInt32(peerSwarm.nActiveConnection,-1)

	var indexInPeerList int = -1
	var indexInSortedPeerList int = -1
	for i, v := range peerSwarm.SortedPeersIndex{
		if v == peer.peerIndex{
			// saves the index where the peer's index is stored
			indexInSortedPeerList = i
			// saves the index of the peer that needs to be removed
			indexInPeerList  = v
			break
		}
	}

	if indexInSortedPeerList >= 0{
		peerSwarm.SortedPeersIndex = append(peerSwarm.SortedPeersIndex[:indexInSortedPeerList], peerSwarm.SortedPeersIndex[indexInSortedPeerList+1:]...)

		peerSwarm.Peers = append(peerSwarm.Peers[:indexInPeerList],peerSwarm.Peers[indexInPeerList+1:]...)
	}
	peerSwarm.peerMutex.Unlock()

}
func (peerSwarm *PeerSwarm) addNewPeer(peerIp ,peerPort,peerID  string )*Peer{
	peerDict := new(parser.Dict)
	peerDict.MapString = make(map[string]string)
	peerDict.MapString["ip"] = peerIp
	peerDict.MapString["port"] = peerPort
	peerDict.MapString["peer id"] =peerID
	newPeer := peerSwarm.NewPeer(peerDict)
	peerSwarm.peerMutex.Lock()
	peerSwarm.PeersMap[newPeer.id] = newPeer
	peerSwarm.Peers = append(peerSwarm.Peers, newPeer)
	peerSwarm.SortedPeersIndex = append(peerSwarm.SortedPeersIndex, len(peerSwarm.Peers)-1)
	atomic.AddInt32(peerSwarm.nActiveConnection,1)
	peerSwarm.peerMutex.Unlock()
	return newPeer

}
func (peerSwarm *PeerSwarm) Len() int {
	return len(peerSwarm.Peers)
}

func (peerSwarm *PeerSwarm) Less(i, j int) bool {
	peerSwarm.peerMutex.Lock()
	ans := peerSwarm.Peers[peerSwarm.SortedPeersIndex[i]].DownloadRate < peerSwarm.Peers[peerSwarm.SortedPeersIndex[j]].DownloadRate
	peerSwarm.peerMutex.Unlock()
	return ans
}

func (peerSwarm *PeerSwarm) Swap(i, j int) {
	temp := peerSwarm.SortedPeersIndex[i]
	peerSwarm.SortedPeersIndex[i] = peerSwarm.SortedPeersIndex[j]
	peerSwarm.SortedPeersIndex[j] = temp
}

type Peer struct {
	ip                string
	port              string
	id                string
	peerIndex 		int
	peerChocking      bool
	peerInterested    bool
	chocked           bool
	interested        bool
	AvailablePieces   []bool
	numByteDownloaded int
	time              float64
	lastTimeStamp     time.Time
	DownloadRate      float64
	connection        *net.TCPConn
}
func (peer *Peer) ConnectTo(peerSwarm *PeerSwarm) {

	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, connectionErr := net.DialTCP("tcp", nil, remotePeerAddr)
	if connectionErr == nil {
		println("---------------------------------------------")
		fmt.Printf("connected to peer at : %v from %v\n", remotePeerAddr.String(), connection.LocalAddr().String())

		msg := MSG{MsgID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.File.infoHash), MyPeerID: utils.MyID}
		nByteWritten, writeErr := connection.Write(GetMsg(msg))
		fmt.Printf("write %v bytes with %v connectionErr\n", nByteWritten, writeErr)

		handshakeBytes := make([]byte, 68)
		_, readFromConError := bufio.NewReader(connection).Read(handshakeBytes)

		fmt.Printf("Back HandShake %v\n", handshakeBytes)

		_, handShakeErr := ParseHandShake(handshakeBytes, peerSwarm.torrent.File.infoHash)

		if handShakeErr == nil && readFromConError == nil {
			// if handshake is successful
			// we attached the connection to the peer struct
			peer.connection = connection
			peerSwarm.activeConnection[remotePeerAddr.String()] = peer
			defer peerSwarm.DropConnection(peer)

			i := 0
			_, _ = connection.Write(GetMsg(MSG{MsgID: InterestedMsg}))

			for readFromConError == nil {
				incomingMsg := make([]byte, 32000)

				_, readFromConError = bufio.NewReader(connection).Read(incomingMsg)

				parsedMsg, parserMsgErr := ParseMsg(incomingMsg, peer)

				if parserMsgErr != nil {
					_ = connection.Close()
					break
				}

				/// TODO will probably use a worker pool here !
				peerSwarm.torrent.msgRouter(parsedMsg)

				fmt.Printf("msg# %v \nparsed msg \n msg ID %v LEN %v\n", i, parsedMsg.MsgID, parsedMsg.MsgLen)

				i++
				println("---------------------------------------------")

			}

		} else {
			fmt.Printf("%v\n", handShakeErr)
		}
	} else {
		fmt.Printf("%v\n", connectionErr)
	}
}
func (peer *Peer) updateState(choked, interested bool,torrent *Torrent) {

	peer.chocked = choked
	peer.interested = interested

	if peer.interested {
		torrent.PeerSwarm.interestedPeerMap[peer.id] = peer
	} else {
		delete(torrent.PeerSwarm.interestedPeerMap, peer.id)
		// if peer isn't interested , no need to leave it  unchoked
		for i, v := range torrent.PeerSwarm.unChockedPeer {
			if peer.id == v.id {
				torrent.PeerSwarm.unChockedPeer[i] = nil
			}
		}
	}
}
