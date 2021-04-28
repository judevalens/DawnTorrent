package PeerProtocol

import (
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/maps/hashmap"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"

	//"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	AddPeer                = 0
	ConnectToPeer = 8
	IncomingConnection = 9
	RemovePeer             = 1
	AddActivePeer          = 4
	SortPeerByDownloadRate = 3
	RemovePendingRequest   = 5
	isPeerFree             = 2
	startReceivingIncomingConnection = 6
	stopReceivingIncomingConnection = 7
)

type PeerSwarm struct {
	PeerSorter  *PeerSorter
	Peers                       []*Peer
	PeersMap                    *hashmap.Map
	PeerByDownloadRateTimeStamp time.Time
	PeerByDownloadRateMutex     *sync.RWMutex
	interestedPeerIndex         []int
	interestedPeerMap           map[string]*Peer
	interestingPeer             map[string]*Peer
	unChockedPeer               []*Peer
	unChockedPeerMap                map[string]*Peer
	peerMutex                       *sync.RWMutex
	interestingPeerMutex            sync.Mutex
	activeConnectionMutex           *sync.RWMutex
	activeConnection                *hashmap.Map
	maxConnection                   int
	torrent                         *Torrent
	trackerInterval                 int
	peerOperation                   chan *peerOperation
	lastSelectedPeer                int
	stopReceivingIncomingConnection chan bool
	receivingIncomingConnection     bool
}

func NewPeerSwarm(torrent *Torrent) *PeerSwarm {
	newPeerSwarm := new(PeerSwarm)
	newPeerSwarm.PeerSorter = new(PeerSorter)
	newPeerSwarm.PeerSorter.activePeers = make([]*Peer,0)
	newPeerSwarm.peerMutex = new(sync.RWMutex)
	newPeerSwarm.PeerByDownloadRateMutex = new(sync.RWMutex)
	newPeerSwarm.PeersMap = hashmap.New()
	newPeerSwarm.interestingPeer = make(map[string]*Peer)
	newPeerSwarm.interestedPeerMap = make(map[string]*Peer)
	newPeerSwarm.unChockedPeer = make([]*Peer, 4)
	//newPeerSwarm.unChockedPeerMap = make(map[string]*Peer,0)
	newPeerSwarm.activeConnectionMutex = new(sync.RWMutex)
	newPeerSwarm.activeConnection = hashmap.New()
	newPeerSwarm.maxConnection = 70
	newPeerSwarm.torrent = torrent
	newPeerSwarm.peerOperation = make(chan *peerOperation)
	newPeerSwarm.stopReceivingIncomingConnection = make(chan bool,1)
	go newPeerSwarm.peersManager()
	//go newPeerSwarm.startServer()

	return newPeerSwarm
}

func (peerSwarm *PeerSwarm) Listen() {
	server, err := net.ListenTCP("tcp", utils.LocalAddr2)
	peerSwarm.receivingIncomingConnection  = true
	//fmt.Printf("Listening on %v\n", server.Addr().String())
	if err == nil {
		for atomic.LoadInt32(peerSwarm.torrent.Downloader.State) == StartedState{
			var connection *net.TCPConn
			var connErr error

			select {
			case stopListening := <- peerSwarm.stopReceivingIncomingConnection:
				if stopListening{
					return
				}
			default:
				connection, connErr = server.AcceptTCP()

			}
			if connErr == nil && connection != nil{
				// limits the number of active connection
				//println("newPeer\n")

				peerOperation := new(peerOperation)
				peerOperation.operation = AddActivePeer

				peerSwarm.peerOperation <- peerOperation
				//} else {
				////println(peerSwarm.nActiveConnection)
				//	//println(peerSwarm.maxConnection)
				//	println("max connection reached!!!!!")

			} else {
				_ = connection.Close()
			}

		}

		if server != nil {
			_ = server.Close()
		}
		peerSwarm.receivingIncomingConnection  = true

	}
}

func (peerSwarm *PeerSwarm) connect(peer *Peer) {

	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, connectionErr := net.DialTCP("tcp", nil, remotePeerAddr)
		fmt.Printf("peer addr : %v, con %v\n",remotePeerAddr.String(),connection)
	if connectionErr == nil {
		_ = connection.SetKeepAlive(true)
		_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)
		fmt.Printf("keep ALive %v", utils.KeepAliveDuration)
		msg := MSG{ID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.Downloader.InfoHash), MyPeerID: utils.MyID}
		_, _ = connection.Write(GetMsg(msg, peer).RawMsg)

		handshakeBytes := make([]byte, 68)
		_, readErr := io.ReadFull(connection, handshakeBytes)

		if readErr == nil {
			_, handShakeErr := ParseHandShake(handshakeBytes, peerSwarm.torrent.Downloader.InfoHash)
			if handShakeErr == nil {

				peer.connection = connection
				peerOperation := new(peerOperation)
				peerOperation.operation = AddActivePeer
				peerOperation.peer = peer
				peerSwarm.peerOperation <- peerOperation
				peerSwarm.torrent.jobQueue.AddJob(GetMsg(MSG{ID: UnchockeMsg}, peer))
				peerSwarm.torrent.jobQueue.AddJob(GetMsg(MSG{ID: InterestedMsg}, peer))
				err := peer.receive(connection, peerSwarm)
				peerOperation.operation = RemovePeer
				peerSwarm.peerOperation <- peerOperation
				fmt.Printf("\nconnec err %v\n", err)
				/////os.Exit(22)

			}
		}
	} else {
		//fmt.Printf("connection failed \n")
	}
}

func (peerSwarm *PeerSwarm) handleNewPeer(connection *net.TCPConn) {
	var newPeer *Peer
	_ = connection.SetKeepAlive(true)
	_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)

	handShakeMsg := make([]byte, 68)
	_, readErr := io.ReadFull(connection, handShakeMsg)
	if readErr == nil {
		parsedHandShakeMsg, handShakeErr := ParseHandShake(handShakeMsg, peerSwarm.torrent.Downloader.InfoHash)

		if handShakeErr == nil {
			remotePeerAddr, _ := net.ResolveTCPAddr("tcp", connection.RemoteAddr().String())
			handShakeMsgResponse := GetMsg(MSG{ID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.Downloader.InfoHash), MyPeerID: utils.MyID}, nil)
			_, writeErr := connection.Write(handShakeMsgResponse.RawMsg)
			if writeErr == nil {
				newPeer = peerSwarm.newPeerFromStrings(remotePeerAddr.IP.String(), strconv.Itoa(remotePeerAddr.Port), parsedHandShakeMsg.peerID)
				newPeer.connection = connection
				peerOperation := new(peerOperation)
				peerOperation.operation = AddPeer
				peerOperation.peer = newPeer

				peerSwarm.peerOperation <- peerOperation
				peerOperation.operation = AddActivePeer
				peerSwarm.peerOperation <- peerOperation

				peerSwarm.torrent.jobQueue.AddJob(GetMsg(MSG{ID: InterestedMsg}, newPeer))
				_ = newPeer.receive(connection, peerSwarm)
				peerOperation.operation = RemovePeer
				peerSwarm.peerOperation <- peerOperation

			} else {

				fmt.Printf("err %v", writeErr)
				//os.Exit(9933)
			}

		} else {
			fmt.Printf("err %v", handShakeErr)
			///os.Exit(9933)
		}

	} else {
		_ = connection.Close()
	}

}

func (peerSwarm *PeerSwarm) createPeersFromTracker(peers *parser.BMap) {
	peerSwarm.Peers = make([]*Peer, 0)
	_, isPresent := peers.BLists["peers"]
	if isPresent {
		for _, peerDict := range peers.BLists["peers"].BMaps {
			peer := peerSwarm.newPeerFromStrings(peerDict.Strings["ip"], peerDict.Strings["port"], peerDict.Strings["peer id"])
			fmt.Printf("peer ip : %v, peer id %v\n",peerDict.Strings["ip"],peerDict.Strings["peer id"])
			go peerSwarm.connect(peer)
		}
	} else {

		peers := peers.Strings["peers"]
		i := 0
		for i < len(peers) {
			ip, port, id := peerSwarm.newPeerFromBytes([]byte(peers[i:(i + 6)]))
			peer := peerSwarm.newPeerFromStrings(ip, port, id)
			go peerSwarm.connect(peer)
			i += 6
		}

	}

}

func (peerSwarm *PeerSwarm) httpTrackerRequest(state int) (*parser.BMap, error) {
	var err error
	PeerId := utils.MyID
	infoHash := peerSwarm.torrent.Downloader.InfoHash

	event := ""
	switch state {
	case 0:
		event = "started"
	case 1:
		event = "stopped"
	case 2:
		event = "completed"

	}

	//fmt.Printf("%x\n", InfoHash)

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", infoHash)
	trackerRequestParam.Add("peer_id", PeerId)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(peerSwarm.torrent.Downloader.uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(peerSwarm.torrent.Downloader.TotalDownloaded))
	trackerRequestParam.Add("left", strconv.Itoa(peerSwarm.torrent.Downloader.left))
	trackerRequestParam.Add("event", event)

	trackerUrl := peerSwarm.torrent.Downloader.Announce + "?" + trackerRequestParam.Encode()
	fmt.Printf("\n Param \n %v \n", trackerUrl)

	trackerResponseByte, _ := http.Get(trackerUrl)
	trackerResponse, _ := ioutil.ReadAll(trackerResponseByte.Body)

	fmt.Printf("%v\n", string(trackerResponse))

	trackerDictResponse := parser.UnmarshallFromArray(trackerResponse)

	_, isPresent := trackerDictResponse.Strings["failure reason"]

	if isPresent {
		err = errors.New(trackerDictResponse.Strings["failure reason"])

	}

	return trackerDictResponse, err

}

func (peerSwarm *PeerSwarm) udpTrackerRequest(event int) (UdpMSG, error) {
	// TODO
	// this needs to be moved
	ipByte := make([]byte, 0)
	start := 0
	end := 0
	// just a little fix to know when the we reach the last number of an IP
	ipString := utils.LocalAddr.IP.String() + "."
	for _, c := range ipString {
		if c == '.' {
			p, _ := strconv.Atoi(ipString[start:end])
			ipByte = append(ipByte, byte(p))
			end++
			start = end
		} else {
			end++
		}

	}
	/////////////////////////////////////////

	cleanedUdpAddr := peerSwarm.torrent.Downloader.Announce[6:]
	trackerAddress, _ := net.ResolveUDPAddr("udp", cleanedUdpAddr)

	incomingMsg := make([]byte, 1024)
	var announceResponse UdpMSG
	var announceResponseErr error
	var succeeded bool

	udpConn, udpErr := net.DialUDP("udp", nil, trackerAddress)
	if udpErr == nil {
		randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
		randN := randSeed.Uint32()
		connectionRequest := udpTrackerConnectMsg(UdpMSG{connectionID: udpProtocolID, action: udpConnectRequest, transactionID: int(randN)})

		_, writingErr := udpConn.Write(connectionRequest)

		if writingErr == nil {
			var readingErr error = nil
			connectionTimeCounter := 0
			nByteRead := 0
			connectionSucceed := false
			var connectionResponse UdpMSG
			var connectionResponseErr error = nil

			// waiting for conn response
			for !connectionSucceed && connectionTimeCounter < 4 {
				//fmt.Printf("counter %v\n", connectionTimeCounter)
				_ = udpConn.SetReadDeadline(time.Now().Add(time.Second * 15))
				nByteRead, readingErr = bufio.NewReader(udpConn).Read(incomingMsg)

				if readingErr != nil {
					//println("time out!!!")
					_, writingErr = udpConn.Write(connectionRequest)
					connectionTimeCounter++
				} else {

					connectionSucceed = true
					connectionResponse, connectionResponseErr = parseUdpTrackerResponse(incomingMsg, nByteRead)
				}
				//fmt.Printf("connection response %v nByteRead %v", incomingMsg, nByteRead)

				if connectionSucceed && connectionResponseErr == nil {

					transactionID := randSeed.Uint32()
					key := randSeed.Uint32()
					announceMsg := udpTrackerAnnounceMsg(UdpMSG{connectionID: connectionResponse.connectionID, action: udpAnnounceRequest, transactionID: int(transactionID), infoHash: []byte(peerSwarm.torrent.Downloader.InfoHash), peerID: []byte(utils.MyID), downloaded: peerSwarm.torrent.Downloader.TotalDownloaded, uploaded: peerSwarm.torrent.Downloader.uploaded, left: peerSwarm.torrent.Downloader.left, event: int(atomic.LoadInt32(peerSwarm.torrent.Downloader.State)), ip: ipByte, key: int(key), port: utils.LocalAddr.Port, numWant: -1})

					_, writingErr = udpConn.Write(announceMsg)

					if writingErr == nil {

						hasResponse := false
						announceRequestCountDown := 0
						for !hasResponse && announceRequestCountDown < 4 {
							_ = udpConn.SetReadDeadline(time.Now().Add(time.Second * 15))
							nByteRead, readingErr = bufio.NewReader(udpConn).Read(incomingMsg)

							if readingErr != nil {
								//println("time out!!!")
								_, writingErr = udpConn.Write(announceMsg)
								announceRequestCountDown++
							} else {
								//fmt.Printf("\n response \n %v", incomingMsg)

								hasResponse = true
								announceResponse, announceResponseErr = parseUdpTrackerResponse(incomingMsg, nByteRead)

								if announceResponseErr != nil {
									succeeded = true
								}
							}

						}

					}

					//fmt.Printf("\n Annouce msg byte \n %v \n peer Len %v", announceMsg,len(announceResponse.peersAddresses))

				}

			}

		} else {
			//	fmt.Printf("writing errors")
		}
	}
	//fmt.Printf("\n response \n %v", announceResponse)

	if succeeded {
		return announceResponse, nil
	} else {
		return UdpMSG{}, errors.New("request to tracker was unsuccessful. please, try again")
	}
}

func (peerSwarm *PeerSwarm) tracker() {

	//determines which protocol the tracker uses
	var peersFromTracker *parser.BMap
	var trackerErr error

	if peerSwarm.torrent.Downloader.Announce[0] == 'u' {
		var udpTrackerResponse UdpMSG
		udpTrackerResponse, trackerErr = peerSwarm.udpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.Downloader.State)))
		peersFromTracker = new(parser.BMap)
		peersFromTracker.Strings = make(map[string]string)
		if trackerErr == nil {
			peersFromTracker.Strings["peers"] = string(udpTrackerResponse.PeersAddresses)
			peerSwarm.trackerInterval = udpTrackerResponse.Interval
		}
	} else if peerSwarm.torrent.Downloader.Announce[0] == 'h' {
		peersFromTracker, trackerErr = peerSwarm.httpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.Downloader.State)))
		if peersFromTracker != nil {
			peerSwarm.trackerInterval, _ = strconv.Atoi(peersFromTracker.Strings["interval"])
		}
	}

	if trackerErr == nil {
		peerSwarm.createPeersFromTracker(peersFromTracker)
		randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomPeer := randomSeed.Perm(len(peerSwarm.Peers))
		maxPeer := math.Ceil((5 / 5.0) * float64(len(peerSwarm.Peers)))
		//fmt.Printf("max peer %v", maxPeer)

		for i := 0; i < int(maxPeer); i++ {
			go peerSwarm.connect(peerSwarm.Peers[randomPeer[i]])
		}
	} else {
		log.Fatal("tracker PieceRequest failed")
	}

}

func (peerSwarm *PeerSwarm) trackerRegularRequest(periodic *periodicFunc) {
	println("sending to tracker ................")
	if atomic.LoadInt32(peerSwarm.torrent.Downloader.State) == StartedState && (time.Now().Sub(periodic.lastExecTimeStamp).Seconds() > float64(peerSwarm.trackerInterval) || periodic.byPassInterval == true) {
		periodic.byPassInterval = false

		var peersFromTracker *parser.BMap
		var trackerErr error

		if peerSwarm.torrent.Downloader.Announce[0] == 'u' {
			var udpTrackerResponse UdpMSG
			udpTrackerResponse, trackerErr = peerSwarm.udpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.Downloader.State)))
			peersFromTracker = new(parser.BMap)
			peersFromTracker.Strings = make(map[string]string)
			if trackerErr == nil {
				peersFromTracker.Strings["peers"] = string(udpTrackerResponse.PeersAddresses)
				peerSwarm.trackerInterval = udpTrackerResponse.Interval
			}
		} else if peerSwarm.torrent.Downloader.Announce[0] == 'h' {
			peersFromTracker, trackerErr = peerSwarm.httpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.Downloader.State)))
			if peersFromTracker != nil {
				peerSwarm.trackerInterval, _ = strconv.Atoi(peersFromTracker.Strings["interval"])
			}
		}



		// if the request to the tracker was successful, we initiate a connections to the peers received
		if trackerErr == nil {
			peerSwarm.createPeersFromTracker(peersFromTracker)
		}

		periodic.lastExecTimeStamp = time.Now()
	}
	fmt.Printf("current %v interval", peerSwarm.trackerInterval)
	fmt.Printf("n peer : %v",len(peerSwarm.PeerSorter.activePeers))

}

func (peerSwarm *PeerSwarm) killSwarm() {


}

func (peerSwarm *PeerSwarm) peersManager() {

	for {
		var pO *peerOperation
		pO = <-peerSwarm.peerOperation
		switch pO.operation {
		case AddPeer:
			peerSwarm.addNewPeer(pO.peer)
		case ConnectToPeer :
			os.Exit(29)
			if len(peerSwarm.PeerSorter.activePeers) < peerSwarm.maxConnection{
				go peerSwarm.connect(pO.peer)
			}
		case IncomingConnection :
			if len(peerSwarm.PeerSorter.activePeers) < peerSwarm.maxConnection{
				go peerSwarm.handleNewPeer(pO.connection)
			}
		case RemovePeer:
			peerSwarm.DropConnection(pO.peer)
		case SortPeerByDownloadRate:
			peerSwarm.SortPeerByDownloadRate()
		case AddActivePeer:
			peerSwarm.addActivePeer(pO.peer)
		case isPeerFree:
			var availablePeer *Peer


			for  _,peer := range peerSwarm.PeerSorter.activePeers{
				if peer != nil{
					if peer.isPeerFree() {
						availablePeer = peer
						break
					}
				}
			}

			println("requesting tracker ...........")
			if availablePeer == nil && len(peerSwarm.PeerSorter.activePeers) < 2{
				peerSwarm.torrent.LifeCycleChannel <- sendTrackerRequest
			}

			pO.freePeerChannel <- availablePeer
		case startReceivingIncomingConnection:
			if !peerSwarm.receivingIncomingConnection {
				peerSwarm.receivingIncomingConnection = true

				go peerSwarm.Listen()
			}
		case stopReceivingIncomingConnection:
			if peerSwarm.receivingIncomingConnection {
				go func() {
					peerSwarm.stopReceivingIncomingConnection <- true
				}()
			}
		}
	}

}

func (peerSwarm *PeerSwarm) trackerRequestEvent() {

}
func (peerSwarm *PeerSwarm) NewPeer(dict *parser.BMap) *Peer {
	newPeer := new(Peer)
	newPeer.peerPendingRequest = make([]*PieceRequest, 0)
	newPeer.mutex = new(sync.RWMutex)
	newPeer.id = dict.Strings["peer id"]
	newPeer.port = dict.Strings["port"]
	newPeer.ip = dict.Strings["ip"]
	newPeer.peerIsChocking = true
	newPeer.interested = false
	newPeer.chocked = true
	newPeer.interested = false
	newPeer.AvailablePieces = make([]bool, int(math.Ceil(float64(peerSwarm.torrent.Downloader.nPiece)/8.0)*8))
	newPeer.isFree = true
	return newPeer
}
func (peerSwarm *PeerSwarm) newPeerFromBytes(peer []byte) (string, string, string) {
	newPeer := new(Peer)

	newPeer.AvailablePieces = make([]bool, peerSwarm.torrent.Downloader.nPiece)

	ipByte := peer[0:4]

	ipString := ""
	for i := range ipByte {

		formatByte := make([]byte, 0)
		formatByte = append(formatByte, 0)
		formatByte = append(formatByte, ipByte[i])
		n := binary.BigEndian.Uint16(formatByte)
		//fmt.Printf("num byte %d\n", formatByte)
		//fmt.Printf("num %v\n", n)

		if i != len(ipByte)-1 {
			ipString += strconv.FormatUint(uint64(n), 10) + "."
		} else {
			ipString += strconv.FormatUint(uint64(n), 10)
		}
	}

	portBytes := binary.BigEndian.Uint16(peer[4:6])

	return ipString, strconv.FormatUint(uint64(portBytes), 10), ipString + ":" + newPeer.port
}
func (peerSwarm *PeerSwarm) newPeerFromStrings(peerIp, peerPort, peerID string) *Peer {
	peerDict := new(parser.BMap)
	peerDict.Strings = make(map[string]string)
	peerDict.Strings["ip"] = peerIp
	peerDict.Strings["port"] = peerPort
	peerDict.Strings["peer id"] = peerID
	newPeer := peerSwarm.NewPeer(peerDict)
	return newPeer

}
func (peerSwarm *PeerSwarm) DropConnection(peer *Peer) {
	peer.connection = nil
	peerSwarm.activeConnection.Remove(peer.id)
	nActivePeer := len(peerSwarm.PeerSorter.activePeers)
	for p := 0; p < nActivePeer; p++ {



		currentPeer := peerSwarm.PeerSorter.activePeers[p]
		if currentPeer.id == peer.id {

			peerSwarm.PeerSorter.activePeers[p],peerSwarm.PeerSorter.activePeers[nActivePeer-1]= peerSwarm.PeerSorter.activePeers[nActivePeer-1],nil

			peerSwarm.PeerSorter.activePeers = peerSwarm.PeerSorter.activePeers[:nActivePeer-1]

			break
		}

	}

	//peer = nil

}

func (peerSwarm *PeerSwarm) addNewPeer(peer *Peer) *Peer {
	peerSwarm.PeersMap.Put(peer.id, peer)
	peerSwarm.Peers = append(peerSwarm.Peers, peer)
	peer.peerIndex = len(peerSwarm.Peers) - 1
	return peer
}
func (peerSwarm *PeerSwarm) addActivePeer(peer *Peer) {
	peerSwarm.activeConnection.Put(peer.id, peer)
	peerSwarm.PeerSorter.activePeers = append(peerSwarm.PeerSorter.activePeers,peer)
	os.Exit(222)
}

func (peerSwarm *PeerSwarm) SortPeerByDownloadRate() {
	if time.Now().Sub(peerSwarm.PeerByDownloadRateTimeStamp) >= peerDownloadRatePeriod {
		sort.Sort(peerSwarm.PeerSorter)
		peerSwarm.PeerByDownloadRateTimeStamp = time.Now()
	}
}

type PeerSorter struct {
	activePeers []*Peer
}

func(PeerSorter *PeerSorter) Len() int{
	return len(PeerSorter.activePeers)
}

func(PeerSorter *PeerSorter) Less(i,j int) bool{

	return !(PeerSorter.activePeers[i].DownloadRate < PeerSorter.activePeers[j].DownloadRate)
}

func(PeerSorter *PeerSorter) Swap(i,j int) {
	PeerSorter.activePeers[i],PeerSorter.activePeers[j] = PeerSorter.activePeers[j],PeerSorter.activePeers[i]
}

type Peer struct {
	ip                              string
	port                            string
	id                              string
	peerIndex                       int
	peerIsChocking                  bool
	peerIsInteresting               bool
	chocked                         bool
	interested                      bool
	AvailablePieces                 []bool
	numByteDownloaded               int
	time                            time.Time
	lastTimeStamp                   time.Time
	DownloadRate                    float64
	connection                      *net.TCPConn
	mutex                           *sync.RWMutex
	peerPendingRequest              []*PieceRequest
	lastPeerPendingRequestTimeStamp time.Time
	isFree                          bool
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
	var readFromConnError error
	i := 0
	msgLenBuffer := make([]byte, 4)

	for readFromConnError == nil {

		var nByteRead int
		nByteRead, readFromConnError = io.ReadFull(connection, msgLenBuffer)
		msgLen := int(binary.BigEndian.Uint32(msgLenBuffer[0:4]))
		incomingMsgBuffer := make([]byte, msgLen)

		nByteRead, readFromConnError = io.ReadFull(connection, incomingMsgBuffer)

		parsedMsg, parserMsgErr := ParseMsg(bytes.Join([][]byte{msgLenBuffer, incomingMsgBuffer}, []byte{}), peer)

		if parserMsgErr == nil {
			// TODO will probably use a worker pool here !
			//peerSwarm.DawnTorrent.requestQueue.addJob(parsedMsg)
			if parsedMsg.ID == PieceMsg {
				//peerSwarm.DawnTorrent.pieceQueue.Add(parsedMsg)
				if nByteRead != parsedMsg.Length && nByteRead > 13 {
					fmt.Printf("nByteRead bBytesRead %v, msgLen %v", nByteRead, parsedMsg.Length)
					//os.Exit(25)
				}
				peerSwarm.torrent.msgRouter(parsedMsg)
			} else {
				peerSwarm.torrent.jobQueue.AddJob(parsedMsg)

			}
			//peerSwarm.DawnTorrent.msgRouter(parsedMsg)

			if parsedMsg.ID == UnchockeMsg {
				//os.Exit(213)
			}

		}

		i++
		//println("---------------------------------------------")

	}

	fmt.Printf("dropping peer ..\n")

	return readFromConnError
}
func (peer *Peer) send(msg MSG) error {
	_, writeError := peer.connection.Write(msg.RawMsg)
	return writeError
}

/*
func(peerSwarm *PeerSwarm) msgAssembler(msg []byte,msgStatus,nReadBytes int)(*MSG,int,int){
	if msgStatus == 0 {
		msgLen := int(binary.BigEndian.Uint32(msg[0:4]))
	}
}
*/

func (peer *Peer) isPeerFree() bool {
	numOfExpiredRequest := 0
	peer.mutex.Lock()
	nPendingRequest := len(peer.peerPendingRequest)

	if peer.peerIsChocking {
		//	println("peer is choking")
		peer.mutex.Unlock()
		peer.isFree = false
		return peer.isFree
	}

	if nPendingRequest < 1 {
		peer.isFree = true
	} else {
		for r := nPendingRequest - 1; r >= 0; r-- {
			pendingRequest := peer.peerPendingRequest[r]

			if time.Now().Sub(pendingRequest.timeStamp).Seconds() >= maxWaitingTime.Seconds() {
				peer.peerPendingRequest[r], peer.peerPendingRequest[len(peer.peerPendingRequest)-1] = peer.peerPendingRequest[len(peer.peerPendingRequest)-1], nil

				peer.peerPendingRequest = peer.peerPendingRequest[:len(peer.peerPendingRequest)-1]

				numOfExpiredRequest++
			}

		}

		if numOfExpiredRequest >= 1 {
			peer.isFree = true
		} else {
			peer.isFree = false
		}
	}

	peer.mutex.Unlock()
	return peer.isFree
}

type peerOperation struct {
	operation int
	peer      *Peer
	// used to get an available peer
	freePeerChannel chan *Peer
	connection *net.TCPConn
}
