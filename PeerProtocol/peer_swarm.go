package PeerProtocol

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/maps/hashmap"
	"io"
	"io/ioutil"
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
	Peers                   []*Peer
	PeersMap                *hashmap.Map
	PeerByDownloadRate      []*Peer
	PeerByDownloadRateMutex *sync.RWMutex
	interestedPeerIndex     []int
	interestedPeerMap       map[string]*Peer
	interestingPeer         map[string]*Peer
	unChockedPeer           []*Peer
	unChockedPeerMap        map[string]*Peer
	peerMutex               *sync.RWMutex
	interestingPeerMutex    sync.Mutex
	activeConnectionMutex   *sync.RWMutex
	activeConnection        *hashmap.Map
	maxConnection           int32
	nActiveConnection       *int32

	torrent *Torrent
}

func NewPeerSwarm(torrent *Torrent) *PeerSwarm {
	newPeerSwarm := new(PeerSwarm)
	newPeerSwarm.peerMutex = new(sync.RWMutex)
	newPeerSwarm.PeerByDownloadRateMutex = new(sync.RWMutex)
	newPeerSwarm.PeersMap = hashmap.New()
	newPeerSwarm.interestingPeer = make(map[string]*Peer)
	newPeerSwarm.interestedPeerMap = make(map[string]*Peer)
	newPeerSwarm.unChockedPeer = make([]*Peer, 4)
	//newPeerSwarm.unChockedPeerMap = make(map[string]*Peer,0)
	newPeerSwarm.activeConnectionMutex = new(sync.RWMutex)
	newPeerSwarm.activeConnection = hashmap.New()
	newPeerSwarm.maxConnection = 1000
	newPeerSwarm.nActiveConnection = new(int32)
	atomic.AddInt32(newPeerSwarm.nActiveConnection, 0)
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
				if *peerSwarm.nActiveConnection <= peerSwarm.maxConnection {
					println("newPeer\n")

					go peerSwarm.handleNewPeer(connection)
				} else {
					//println(peerSwarm.nActiveConnection)
					//	println(peerSwarm.maxConnection)
					///println("max connection reached!!!!!")
				}
			} else {
				_ = connection.Close()
			}
		} else {
			//println("eer")
			//log.Fatal(err)
		}

	}
}
func (peerSwarm *PeerSwarm) connect(peer *Peer) {

	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, connectionErr := net.DialTCP("tcp", nil, remotePeerAddr)

	if connectionErr == nil {
		_ = connection.SetKeepAlive(true)
		_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)

		msg := MSG{MsgID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.File.infoHash), MyPeerID: utils.MyID}
		_, _ = connection.Write(GetMsg(msg, peer).rawMsg)

		handshakeBytes := make([]byte, 68)
		_, readErr := io.ReadFull(connection, handshakeBytes)

		if readErr == nil {
			_, handShakeErr := ParseHandShake(handshakeBytes, peerSwarm.torrent.File.infoHash)
			if handShakeErr == nil {
				defer peerSwarm.DropConnection(peer)
				peerSwarm.peerMutex.Lock()
				peer.connection = connection
				peerSwarm.activeConnection.Put(peer.id, peer)
				atomic.AddInt32(peerSwarm.nActiveConnection, 1)
				peerSwarm.peerMutex.Unlock()
				_ = peer.receive(connection, peerSwarm)
			}
		}
	} else {
		fmt.Printf("connection failed \n")
	}
}

func (peerSwarm *PeerSwarm) handleNewPeer(connection *net.TCPConn) {
	var newPeer *Peer
	_ = connection.SetKeepAlive(true)
	_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)

	handShakeMsg := make([]byte, 68)
	_, readErr := io.ReadFull(connection, handShakeMsg)

	if readErr == nil {
		parsedHandShakeMsg, handShakeErr := ParseHandShake(handShakeMsg, peerSwarm.torrent.File.infoHash)

		if handShakeErr == nil {
			remotePeerAddr, _ := net.ResolveTCPAddr("tcp", connection.RemoteAddr().String())
			handShakeMsgResponse := GetMsg(MSG{MsgID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.File.infoHash), MyPeerID: utils.MyID}, nil)
			_, writeErr := connection.Write(handShakeMsgResponse.rawMsg)
			if writeErr == nil {
				defer peerSwarm.DropConnection(newPeer)
				newPeer = peerSwarm.addNewPeer(remotePeerAddr.IP.String(), strconv.Itoa(remotePeerAddr.Port), parsedHandShakeMsg.peerID)
				newPeer.connection = connection
				peerSwarm.peerMutex.Lock()
				peerSwarm.activeConnection.Put(newPeer.id, newPeer)
				atomic.AddInt32(peerSwarm.nActiveConnection, 1)
				peerSwarm.peerMutex.Unlock()
				_ = newPeer.receive(connection, peerSwarm)
			}

		} else {
			fmt.Printf("%v\n", handShakeErr)
			fmt.Printf("%v", handShakeMsg)
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
			peerSwarm.addNewPeer(peer.MapString["ip"], peer.MapString["port"], peer.MapString["peer id"])
		}
	} else {

		peers := peers.MapString["peers"]
		i := 0
		fmt.Printf("LEN %v \n", len(peers))
		for i < len(peers) {
			fmt.Printf("peers : %v\n", []byte(peers[i:(i+6)]))

			ip, port, id := peerSwarm.NewPeerFromString([]byte(peers[i:(i + 6)]))
			peerSwarm.addNewPeer(ip, port, id)

			i += 6
		}

	}

}

func (peerSwarm *PeerSwarm) TrackerRequest(event string) (*parser.Dict, error) {
	var err error
	PeerId := utils.MyID
	infoHash := peerSwarm.torrent.File.infoHash
	uploaded := 0
	downloaded := 0

	fmt.Printf("%x\n", infoHash)

	left := peerSwarm.torrent.File.fileLen

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

	return trackerDictResponse, err

}

func (peerSwarm *PeerSwarm) initConnection() {

	peersFromTracker, err := peerSwarm.TrackerRequest(startedEvent)

	if err == nil {
		peerSwarm.addPeersFromTracker(peersFromTracker)
		randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomPeer := randomSeed.Perm(len(peerSwarm.Peers))
		maxPeer := math.Ceil((5 / 5.0) * float64(len(peerSwarm.Peers)))
		fmt.Printf("max peer %v", maxPeer)

		for i := 0; i < int(maxPeer); i++ {
			go peerSwarm.connect(peerSwarm.Peers[randomPeer[i]])
		}
	}

	go peerSwarm.Listen()

}
func (peerSwarm *PeerSwarm) NewPeer(dict *parser.Dict) *Peer {
	newPeer := new(Peer)
	newPeer.id = dict.MapString["peer id"]
	newPeer.port = dict.MapString["port"]
	newPeer.ip = dict.MapString["ip"]
	newPeer.peerIsChocking = true
	newPeer.interested = false
	newPeer.chocked = true
	newPeer.interested = false
	newPeer.AvailablePieces = make([]bool, int(math.Ceil(float64(peerSwarm.torrent.File.nPiece)/8.0)*8))

	return newPeer
}
func (peerSwarm *PeerSwarm) NewPeerFromString(peer []byte) (string, string, string) {
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

	return ipString, strconv.FormatUint(uint64(portBytes), 10), ipString + ":" + newPeer.port
}
func (peerSwarm *PeerSwarm) DropConnection(peer *Peer) {
	peerSwarm.peerMutex.Lock()
	//peerSwarm.activeConnection.Remove(peer.id)
	atomic.AddInt32(peerSwarm.nActiveConnection, -1)
	peerSwarm.peerMutex.Unlock()
}
func (peerSwarm *PeerSwarm) addNewPeer(peerIp, peerPort, peerID string) *Peer {
	peerDict := new(parser.Dict)
	peerDict.MapString = make(map[string]string)
	peerDict.MapString["ip"] = peerIp
	peerDict.MapString["port"] = peerPort
	peerDict.MapString["peer id"] = peerID
	newPeer := peerSwarm.NewPeer(peerDict)
	peerSwarm.peerMutex.Lock()
	peerSwarm.PeersMap.Put(newPeer.id, newPeer)
	peerSwarm.Peers = append(peerSwarm.Peers, newPeer)
	newPeer.peerIndex = len(peerSwarm.Peers) - 1
	peerSwarm.peerMutex.Unlock()
	return newPeer

}
func (peerSwarm *PeerSwarm) Len() int {
	return len(peerSwarm.Peers)
}
func (peerSwarm *PeerSwarm) Less(i, j int) bool {
	peerSwarm.peerMutex.Lock()
	ans := peerSwarm.Peers[0].DownloadRate < peerSwarm.Peers[0].DownloadRate
	peerSwarm.peerMutex.Unlock()
	return ans
}
func (peerSwarm *PeerSwarm) Swap(i, j int) {

}

func (peerSwarm *PeerSwarm) UpdatePeerByDownloadRate(peer *Peer) {
	for i := len(peerSwarm.PeerByDownloadRate) - 1; i <= 0; i-- {
		if peer.DownloadRate > peerSwarm.PeerByDownloadRate[i].DownloadRate {

			peerSwarm.PeerByDownloadRateMutex.Lock()
			peerSwarm.PeerByDownloadRate[i] = peer
			peerSwarm.PeerByDownloadRateMutex.Unlock()

			break
		}
	}
}

func (peerSwarm *PeerSwarm) GetPeerByDownloadRate(index int) *Peer {
	p := peerSwarm.PeerByDownloadRate[index]

	i := 0

	for p == nil && i > len(peerSwarm.PeerByDownloadRate) {
		peerSwarm.peerMutex.RLock()
		p = peerSwarm.PeerByDownloadRate[i]
		peerSwarm.peerMutex.RUnlock()

	}
	randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
	for p == nil && i < len(peerSwarm.Peers) {
		randN := randomSeed.Intn(len(peerSwarm.Peers))
		activeConnectionKey := peerSwarm.activeConnection.Keys()
		activeConnectionKeyS := activeConnectionKey[randN].(string)
		peerSwarm.peerMutex.RLock()
		v, _ := peerSwarm.activeConnection.Get(activeConnectionKeyS)
		peerSwarm.peerMutex.RUnlock()

		p = v.(*Peer)

		//making sure that the peer isn't already in the list
		for _, v := range peerSwarm.PeerByDownloadRate {
			if v.id == p.id {
				p = nil
				break
			}
		}
		i++
	}

	return p

}

type Peer struct {
	ip                string
	port              string
	id                string
	peerIndex         int
	peerIsChocking    bool
	peerIsInteresting bool
	chocked           bool
	interested        bool
	AvailablePieces   []bool
	numByteDownloaded int
	time              float64
	lastTimeStamp     time.Time
	DownloadRate      float64
	connection        *net.TCPConn
}

func (peer *Peer) updateState(choked, interested bool, torrent *Torrent) {

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

func (peer *Peer) receive(connection *net.TCPConn, peerSwarm *PeerSwarm) error {
	incomingMsg := make([]byte, 18000)
	var readFromConnError error
	i := 0
	for readFromConnError == nil {

		_, readFromConnError = bufio.NewReader(connection).Read(incomingMsg)

		parsedMsg, parserMsgErr := ParseMsg(incomingMsg, peer)

		if parserMsgErr == nil {
			// TODO will probably use a worker pool here !
			//peerSwarm.torrent.requestQueue.addJob(parsedMsg)
			if parsedMsg.MsgID == BitfieldMsg {
			}
			peerSwarm.torrent.incomingMsgQueue.Add(parsedMsg)
			//peerSwarm.torrent.msgRouter(parsedMsg)

			if parsedMsg.MsgID == BitfieldMsg {
				///os.Exit(23)
			}

		}

		i++
		println("---------------------------------------------")

	}

	return readFromConnError
}
func (peer *Peer) send(msg MSG) error {
	_, writeError := peer.connection.Write(msg.rawMsg)

	return writeError
}
