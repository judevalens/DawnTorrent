package PeerProtocol

import (
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"github.com/emirpasic/gods/maps/hashmap"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	//"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type PeerSwarm struct {
	Peers                   []*Peer
	PeersMap                *hashmap.Map
	PeerByDownloadRate      *arraylist.List
	PeerByDownloadRateTimeStamp time.Time
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
	trackerInterval	int
}

func NewPeerSwarm(torrent *Torrent) *PeerSwarm {
	newPeerSwarm := new(PeerSwarm)
	newPeerSwarm.peerMutex = new(sync.RWMutex)
	newPeerSwarm.PeerByDownloadRateMutex = new(sync.RWMutex)
	newPeerSwarm.PeersMap = hashmap.New()
	newPeerSwarm.PeerByDownloadRate = arraylist.New()
	newPeerSwarm.interestingPeer = make(map[string]*Peer)
	newPeerSwarm.interestedPeerMap = make(map[string]*Peer)
	newPeerSwarm.unChockedPeer = make([]*Peer, 4)
	//newPeerSwarm.unChockedPeerMap = make(map[string]*Peer,0)
	newPeerSwarm.activeConnectionMutex = new(sync.RWMutex)
	newPeerSwarm.activeConnection = hashmap.New()
	newPeerSwarm.maxConnection = 70
	newPeerSwarm.nActiveConnection = new(int32)
	atomic.AddInt32(newPeerSwarm.nActiveConnection, 0)
	newPeerSwarm.torrent = torrent

	/////////newPeerSwarm.initConnection()
	//go newPeerSwarm.trackerRegularRequest()
	//go newPeerSwarm.Listen()
	return newPeerSwarm
}

func (peerSwarm *PeerSwarm) Listen() {
	server, err := net.ListenTCP("tcp", utils.LocalAddr2)
	//fmt.Printf("Listening on %v\n", server.Addr().String())
	if err == nil{
	for atomic.LoadInt32(peerSwarm.torrent.File.Status) == StartedState {
			connection, connErr := server.AcceptTCP()
			if connErr == nil {
				// limits the number of active connection
				//if *peerSwarm.nActiveConnection <= peerSwarm.maxConnection {
					//println("newPeer\n")

					go peerSwarm.handleNewPeer(connection)
				//} else {
					////println(peerSwarm.nActiveConnection)
					//	//println(peerSwarm.maxConnection)
				//	println("max connection reached!!!!!")
			//	}
			} else {
				_ = connection.Close()
			}

	}

	if server != nil{
		_ = server.Close()
	}
	}
}
func (peerSwarm *PeerSwarm) connect(peer *Peer) {

	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, connectionErr := net.DialTCP("tcp", nil, remotePeerAddr)

	if connectionErr == nil {
		_ = connection.SetKeepAlive(true)
		_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)
			fmt.Printf("keep ALive %v",utils.KeepAliveDuration)
		msg := MSG{MsgID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.File.InfoHash), MyPeerID: utils.MyID}
		_, _ = connection.Write(GetMsg(msg, peer).rawMsg)

		handshakeBytes := make([]byte, 68)
		_, readErr := io.ReadFull(connection, handshakeBytes)

		if readErr == nil {
			_, handShakeErr := ParseHandShake(handshakeBytes, peerSwarm.torrent.File.InfoHash)
			if handShakeErr == nil {

				peerSwarm.activeConnectionMutex.Lock()
				peerSwarm.peerMutex.Lock()
				peer.connection = connection
				peerSwarm.activeConnection.Put(peer.id, peer)
				atomic.AddInt32(peerSwarm.nActiveConnection, 1)
				peerSwarm.peerMutex.Unlock()
				peerSwarm.activeConnectionMutex.Unlock()
				peerSwarm.torrent.requestQueue.Add(GetMsg(MSG{MsgID: UnchockeMsg}, peer))


				peerSwarm.torrent.requestQueue.Add(GetMsg(MSG{MsgID: InterestedMsg}, peer))
				peerSwarm.PeerByDownloadRateMutex.Lock()
				peerSwarm.PeerByDownloadRate.Add(peer)
				peerSwarm.PeerByDownloadRateMutex.Unlock()
				err := peer.receive(connection, peerSwarm)
				fmt.Printf("\nconnec err %v\n",err)
				/////os.Exit(22)
				peerSwarm.DropConnection(peer)
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
		parsedHandShakeMsg, handShakeErr := ParseHandShake(handShakeMsg, peerSwarm.torrent.File.InfoHash)

		if handShakeErr == nil {
			remotePeerAddr, _ := net.ResolveTCPAddr("tcp", connection.RemoteAddr().String())
			handShakeMsgResponse := GetMsg(MSG{MsgID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.File.InfoHash), MyPeerID: utils.MyID}, nil)
			_, writeErr := connection.Write(handShakeMsgResponse.rawMsg)
			if writeErr == nil {
				newPeer = peerSwarm.addNewPeer(remotePeerAddr.IP.String(), strconv.Itoa(remotePeerAddr.Port), parsedHandShakeMsg.peerID)
				newPeer.connection = connection
				peerSwarm.activeConnectionMutex.Lock()
				peerSwarm.activeConnection.Put(newPeer.id, newPeer)
				atomic.AddInt32(peerSwarm.nActiveConnection, 1)
				peerSwarm.activeConnectionMutex.Unlock()
				peerSwarm.PeerByDownloadRateMutex.Lock()
				peerSwarm.PeerByDownloadRate.Add(newPeer)
				peerSwarm.PeerByDownloadRateMutex.Unlock()
				peerSwarm.torrent.requestQueue.Add(GetMsg(MSG{MsgID: InterestedMsg}, newPeer))
				peerSwarm.DropConnection(newPeer)
				_ = newPeer.receive(connection, peerSwarm)
			}else{

				fmt.Printf("err %v",writeErr)
				//os.Exit(9933)
			}

		} else {
			fmt.Printf("err %v",handShakeErr)
			///os.Exit(9933)
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
		//fmt.Printf("LEN %v \n", len(peers))
		for i < len(peers) {
		//	fmt.Printf("peers : %v\n", []byte(peers[i:(i+6)]))

			ip, port, id := peerSwarm.NewPeerFromString([]byte(peers[i:(i + 6)]))
			peerSwarm.addNewPeer(ip, port, id)

			i += 6
		}

	}

}

func (peerSwarm *PeerSwarm) httpTrackerRequest(state int) (*parser.Dict, error) {
	var err error
	PeerId := utils.MyID
	infoHash := peerSwarm.torrent.File.InfoHash

	event := ""
	switch state {
	case 0:
		event = "started"
	case 1:
		event ="stopped"
	case 2:
		event = "completed"

	}

	//fmt.Printf("%x\n", InfoHash)


	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", infoHash)
	trackerRequestParam.Add("peer_id", PeerId)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(peerSwarm.torrent.File.uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(peerSwarm.torrent.File.TotalDownloaded))
	trackerRequestParam.Add("left", strconv.Itoa(peerSwarm.torrent.File.left))
	trackerRequestParam.Add("event", event)


	trackerUrl := peerSwarm.torrent.File.Announce + "?" + trackerRequestParam.Encode()
	fmt.Printf("\n Param \n %v \n", trackerUrl)

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

func (peerSwarm *PeerSwarm) udpTrackerRequest(event int) (udpMSG, error) {
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

	cleanedUdpAddr := peerSwarm.torrent.File.Announce[6:]
	trackerAddress, _ := net.ResolveUDPAddr("udp", cleanedUdpAddr)

	incomingMsg := make([]byte, 1024)
	var announceResponse udpMSG
	var announceResponseErr error
	udpConn, udpErr := net.DialUDP("udp", nil, trackerAddress)
	if udpErr == nil {
		randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
		randN := randSeed.Uint32()
		connectionRequest := udpTrackerConnectMsg(udpMSG{connectionID: udpProtocolID, action: udpConnectRequest, transactionID: int(randN)})

		_, writingErr := udpConn.Write(connectionRequest)

		if writingErr == nil {
			var readingErr error = nil
			connectionTimeCounter := 0
			nByteRead := 0
			connectionSucceed := false
			var connectionResponse udpMSG
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
					announceMsg := udpTrackerAnnounceMsg(udpMSG{connectionID: connectionResponse.connectionID, action: udpAnnounceRequest, transactionID: int(transactionID), infoHash: []byte(peerSwarm.torrent.File.InfoHash), peerID: []byte(utils.MyID), downloaded: peerSwarm.torrent.File.TotalDownloaded, uploaded: peerSwarm.torrent.File.uploaded, left: peerSwarm.torrent.File.left, event: int(atomic.LoadInt32(peerSwarm.torrent.File.Status)), ip: ipByte, key: int(key), port: utils.LocalAddr.Port, numWant: -1})

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
							}

						}

					}

					//fmt.Printf("\n Annouce msg byte \n %v \n peer Len %v", announceMsg,len(announceResponse.peersAddresses))

				} else {
					log.Fatal(connectionResponseErr)
				}

			}

		} else {
		//	fmt.Printf("writing errors")
		}
	}
//fmt.Printf("\n response \n %v", announceResponse)
	return announceResponse, announceResponseErr

}

func (peerSwarm *PeerSwarm) tracker() {

	//determines which protocol the tracker uses
var peersFromTracker *parser.Dict
var trackerErr error

	if peerSwarm.torrent.File.Announce[0] == 'u' {
		var udpTrackerResponse udpMSG
		udpTrackerResponse, trackerErr = peerSwarm.udpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.File.Status)))
		peersFromTracker = new(parser.Dict)
		peersFromTracker.MapString = make(map[string]string)
		if trackerErr == nil{
			peersFromTracker.MapString["peers"] = string(udpTrackerResponse.peersAddresses)
			peerSwarm.trackerInterval = udpTrackerResponse.interval
		}
	} else if peerSwarm.torrent.File.Announce[0] == 'h'{
	peersFromTracker, trackerErr = peerSwarm.httpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.File.Status)))
	if peersFromTracker != nil{
		peerSwarm.trackerInterval, _ = strconv.Atoi(peersFromTracker.MapString["interval"])
	}
	}


	if trackerErr == nil {
		peerSwarm.addPeersFromTracker(peersFromTracker)
		randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
		randomPeer := randomSeed.Perm(len(peerSwarm.Peers))
		maxPeer := math.Ceil((5 / 5.0) * float64(len(peerSwarm.Peers)))
		//fmt.Printf("max peer %v", maxPeer)

		for i := 0; i < int(maxPeer); i++ {
			go peerSwarm.connect(peerSwarm.Peers[randomPeer[i]])
		}
	}else{
		log.Fatal("tracker PieceRequest failed")
	}


}

func (peerSwarm *PeerSwarm) trackerRegularRequest(){
	for atomic.LoadInt32(peerSwarm.torrent.File.Status) == StartedState {
		var peersFromTracker *parser.Dict
		var trackerErr error

		if peerSwarm.torrent.File.Announce[0] == 'u' {
			var udpTrackerResponse udpMSG
			udpTrackerResponse, trackerErr = peerSwarm.udpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.File.Status)))
			peersFromTracker = new(parser.Dict)
			peersFromTracker.MapString = make(map[string]string)
			if trackerErr == nil{
				peersFromTracker.MapString["peers"] = string(udpTrackerResponse.peersAddresses)
				peerSwarm.trackerInterval = udpTrackerResponse.interval
			}
		} else if peerSwarm.torrent.File.Announce[0] == 'h'{
			peersFromTracker, trackerErr = peerSwarm.httpTrackerRequest(int(atomic.LoadInt32(peerSwarm.torrent.File.Status)))
			if peersFromTracker != nil{
				peerSwarm.trackerInterval, _ = strconv.Atoi(peersFromTracker.MapString["interval"])
			}
		}

		fmt.Printf("%v interval",peerSwarm.trackerInterval)


		if trackerErr == nil {
			peerSwarm.addPeersFromTracker(peersFromTracker)
			randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
			randomPeer := randomSeed.Perm(len(peerSwarm.Peers))
			maxPeer := math.Ceil((5 / 5.0) * float64(len(peerSwarm.Peers)))
			//fmt.Printf("max peer %v", maxPeer)

			for i := 0; i < int(maxPeer); i++ {
				go peerSwarm.connect(peerSwarm.Peers[randomPeer[i]])
			}
		}else{
			log.Fatal("tracker PieceRequest failed")
		}


		time.Sleep(time.Duration(time.Second.Seconds()*1000000000*float64(peerSwarm.trackerInterval)))
	}
}

func (peerSwarm *PeerSwarm) killSwarm()  {
	peerKeyI := peerSwarm.activeConnection.Keys()


	for _,k := range peerKeyI{
		key := k.(string)
		peerI, found := peerSwarm.activeConnection.Get(key)

		if found{
			peer := peerI.(*Peer)
			peerSwarm.DropConnection(peer)
			fmt.Printf("killing \n")
		}
	}

	peerSwarm.activeConnection.Clear()
	peerSwarm.PeerByDownloadRate.Clear()
}

func (peerSwarm *PeerSwarm) trackerRequestEvent(){

}
func (peerSwarm *PeerSwarm) NewPeer(dict *parser.Dict) *Peer {
	newPeer := new(Peer)
	newPeer.peerPendingRequest = make([]*PieceRequest,0)
	newPeer.peerPendingRequestMutex = new(sync.RWMutex)
	newPeer.id = dict.MapString["peer id"]
	newPeer.port = dict.MapString["port"]
	newPeer.ip = dict.MapString["ip"]
	newPeer.peerIsChocking = true
	newPeer.interested = false
	newPeer.chocked = true
	newPeer.interested = false
	newPeer.AvailablePieces = make([]bool, int(math.Ceil(float64(peerSwarm.torrent.File.nPiece)/8.0)*8))
	newPeer.isFree = true
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
func (peerSwarm *PeerSwarm) DropConnection(peer *Peer) {
	peerSwarm.activeConnectionMutex.Lock()
	peer.connection = nil
	peerSwarm.activeConnection.Remove(peer.id)
	atomic.AddInt32(peerSwarm.nActiveConnection, -1)
	peerSwarm.PeerByDownloadRateMutex.Lock()

	for p:= 0; p < peerSwarm.PeerByDownloadRate.Size(); p++{

		peerInterface , _:=  peerSwarm.PeerByDownloadRate.Get(p)

		peerI := peerInterface.(*Peer)
		if peerI.id == peer.id {
			peerSwarm.PeerByDownloadRate.Remove(p)
			break
		}

	}
	peerSwarm.PeerByDownloadRateMutex.Unlock()

	peerSwarm.activeConnectionMutex.Unlock()
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


func (peerSwarm *PeerSwarm) PeerByDownloadRateComparator (a,b interface{}) int{
	pieceA := a.(*Peer)
	pieceB	:= b.(*Peer)

	switch  {
	case pieceA.DownloadRate > pieceB.DownloadRate:
		return -1
	case pieceA.DownloadRate < pieceB.DownloadRate:
		return 1
	default:
		return 0
	}
}

func (peerSwarm *PeerSwarm) SortPeerByDownloadRate(){
	peerSwarm.PeerByDownloadRateMutex.Lock()
	if time.Now().Sub(peerSwarm.PeerByDownloadRateTimeStamp) >= peerDownloadRatePeriod{
		peerSwarm.PeerByDownloadRate.Sort(peerSwarm.PeerByDownloadRateComparator)
		peerSwarm.PeerByDownloadRateTimeStamp = time.Now()
	}
	peerSwarm.PeerByDownloadRateMutex.Unlock()

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
	time              time.Time
	lastTimeStamp     time.Time
	DownloadRate      float64
	connection        *net.TCPConn
	peerPendingRequestMutex *sync.RWMutex
	peerPendingRequest	[]*PieceRequest
	lastPeerPendingRequestTimeStamp time.Time
	isFree	bool
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
			//peerSwarm.DawnTorrent.requestQueue.addJob(parsedMsg)
			if parsedMsg.MsgID == PieceMsg {
				//peerSwarm.DawnTorrent.pieceQueue.Add(parsedMsg)
				peerSwarm.torrent.msgRouter(parsedMsg)

			}else{
				peerSwarm.torrent.incomingMsgQueue.Add(parsedMsg)

			}
			//peerSwarm.DawnTorrent.msgRouter(parsedMsg)

			if parsedMsg.MsgID == UnchockeMsg {
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
	_, writeError := peer.connection.Write(msg.rawMsg)

	return writeError
}


func(peer *Peer) isPeerFree () bool{
	numOfExpiredRequest := 0
	peer.peerPendingRequestMutex.Lock()
	nPendingRequest := len(peer.peerPendingRequest)

	if peer.peerIsChocking {
	//	println("peer is choking")
		peer.peerPendingRequestMutex.Unlock()
	peer.isFree = false
	return peer.isFree
	}


	if nPendingRequest < 5{
		peer.isFree = true
	}else {
		for r:= nPendingRequest-1; r >= 0; r--{
			pendingRequest := peer.peerPendingRequest[r]

			if time.Now().Sub(pendingRequest.timeStamp).Seconds() >=time.Second.Seconds()*1.5{
				peer.peerPendingRequest[r], peer.peerPendingRequest[len(peer.peerPendingRequest)-1] = peer.peerPendingRequest[len(peer.peerPendingRequest)-1],nil

				peer.peerPendingRequest = peer.peerPendingRequest[:len(peer.peerPendingRequest)-1]

				numOfExpiredRequest++
			}

		}

		if numOfExpiredRequest >= 1{
			peer.isFree = true
		}else{
			peer.isFree = false
		}
	}

	peer.peerPendingRequestMutex.Unlock()
	return peer.isFree
}
