package protocol

import (
	"DawnTorrent/PeerProtocol"
	"DawnTorrent/parser"
	_ "DawnTorrent/parser"
	"DawnTorrent/utils"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/maps/hashmap"
	"io"
	"io/ioutil"
	"log"
	"net"
	_ "net"
	"net/http"
	"net/url"
	_ "os"
	"strconv"
	_ "strconv"
	"sync"
	"time"
)

const (
	AddPeer                          = iota
	ConnectToPeer                    = iota
	IncomingConnection               = iota
	RemovePeer                       = iota
	AddActivePeer                    = iota
	SortPeerByDownloadRate           = iota
	RemovePendingRequest             = iota
	isPeerFree                       = iota
	startReceivingIncomingConnection = iota
	stopReceivingIncomingConnection  = iota
)

const (
	httpTracker = iota
	updTracker = iota
)

type PeerSwarm struct {

	activePeers  					map[string]*Peer
	Peers                           []*Peer
	PeersMap                        map[string]*Peer
	interestedPeerIndex             []int
	interestedPeerMap               map[string]*Peer
	interestingPeer                 map[string]*Peer
	unChockedPeer                   []*Peer
	unChockedPeerMap                map[string]*Peer
	peerMutex                       *sync.RWMutex
	interestingPeerMutex            sync.Mutex
	activeConnectionMutex           *sync.RWMutex
	activeConnection                *hashmap.Map
	nActiveConnection 				int
	maxConnection                   int
	torrent                         *Torrent
	trackerInterval                 int
	peerOperation                   chan PeerOperation2
	lastSelectedPeer                int
	trackerRequestChan  time.Ticker
	initialTrackerRequest  chan interface{}
	stopTrackerRequest chan interface{}
	receivingIncomingConnection     bool
	server *net.TCPListener
}



func (peerSwarm *PeerSwarm) addPeer(peer *Peer) *Peer {
	peerSwarm.PeersMap[peer.id] = peer
	// TODO That's probably redundant
	peerSwarm.Peers = append(peerSwarm.Peers, peer)
	peer.peerIndex = len(peerSwarm.Peers) - 1
	return peer
}
func (peerSwarm *PeerSwarm) handleNewPeer(connection *net.TCPConn) {
	//var newPeer *Peer
	_ = connection.SetKeepAlive(true)
	_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)

	handShakeMsg := make([]byte, 68)
	_, readErr := io.ReadFull(connection, handShakeMsg)
	if readErr == nil {

		// TODO no need to check the info in the parseMethod bc of separation of concern
		_, _ = PeerProtocol.ParseHandShake(handShakeMsg, "peerSwarm.torrent.Downloader.InfoHash")

		/*
			if handShakeErr == nil {
				remotePeerAddr, _ := net.ResolveTCPAddr("tcp", connection.RemoteAddr().String())
				handShakeMsgResponse := GetMsg(MSG{MsgID: HandShakeMsgID, InfoHash: []byte(peerSwarm.torrent.Downloader.InfoHash), MyPeerID: utils.MyID}, nil)
				_, writeErr := connection.Write(handShakeMsgResponse.RawMsg)
				if writeErr == nil {
					newPeer = peerSwarm.newPeerFromStrings(remotePeerAddr.IP.String(), strconv.Itoa(remotePeerAddr.Port), parsedHandShakeMsg.peerID)
					newPeer.connection = connection
					PeerOperation := new(PeerOperation)
					PeerOperation.operation = AddPeer
					PeerOperation.peer = newPeer

					peerSwarm.PeerOperation <- PeerOperation
					PeerOperation.operation = AddActivePeer
					peerSwarm.PeerOperation <- PeerOperation

					peerSwarm.torrent.jobQueue.AddJob(GetMsg(MSG{MsgID: InterestedMsg}, newPeer))
					_ = newPeer.receive(connection, peerSwarm)
					PeerOperation.operation = RemovePeer
					peerSwarm.PeerOperation <- PeerOperation

				} else {

					fmt.Printf("err %v", writeErr)
					//os.Exit(9933)
				}

			} else {
				fmt.Printf("err %v", handShakeErr)
				///os.Exit(9933)
			}
		*/
	} else {
		_ = connection.Close()
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
		msg := PeerProtocol.MSG{MsgID: PeerProtocol.HandShakeMsgID, InfoHash: []byte("peerSwarm.torrent.Downloader.InfoHash"), MyPeerID: utils.MyID}
		_, _ = connection.Write(PeerProtocol.GetMsg(msg, nil).RawMsg)

		handshakeBytes := make([]byte, 68)
		_, readErr := io.ReadFull(connection, handshakeBytes)
		_ = readErr
		/*
			if readErr == nil {
				_, handShakeErr := ParseHandShake(handshakeBytes, peerSwarm.torrent.Downloader.InfoHash)
				if handShakeErr == nil {

					peer.connection = connection
					PeerOperation := new(PeerOperation)
					PeerOperation.operation = AddActivePeer
					PeerOperation.peer = peer
					peerSwarm.PeerOperation <- PeerOperation
					peerSwarm.torrent.jobQueue.AddJob(GetMsg(MSG{MsgID: UnchockeMsg}, peer))
					peerSwarm.torrent.jobQueue.AddJob(GetMsg(MSG{MsgID: InterestedMsg}, peer))
					err := peer.receive(connection, peerSwarm)
					PeerOperation.operation = RemovePeer
					peerSwarm.PeerOperation <- PeerOperation
					fmt.Printf("\nconnec err %v\n", err)
					/////os.Exit(22)

				}
			}*/
	} else {
		//fmt.Printf("connection failed \n")
	}
}
func (peerSwarm *PeerSwarm) DropConnection(peer *Peer) {
	/*
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
*/
}

func (peerSwarm *PeerSwarm) startServer() {
	server, err := net.ListenTCP("tcp", utils.LocalAddr2)
	peerSwarm.server  = server
	//fmt.Printf("Listening on %v\n", server.Addr().String())
	if err == nil {
		for {
			var connection *net.TCPConn
			var connErr error
			connection, connErr = server.AcceptTCP()
			if connErr == nil{

				peerSwarm.peerOperation <- IncommingPeerConnection{
					swarm: peerSwarm,
					conn: connection,
				}
				

			} else {
				log.Fatalf("error: %v", err)

			}

		}

	}else {
		log.Fatalf("error: %v", err)
	}
}

func (peerSwarm *PeerSwarm) sendHTTPTrackerRequest(state int, uploaded, totalDownloaded, left int, infoHash string, announcerURL string) (*parser.BMap, error) {

	myID := utils.MyID

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
	trackerRequestParam.Add("peer_id", myID)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(totalDownloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	trackerUrl := announcerURL + "?" + trackerRequestParam.Encode()
	fmt.Printf("\n Param \n %v \n", trackerUrl)

	trackerResponseByte, _ := http.Get(trackerUrl)
	trackerResponse, _ := ioutil.ReadAll(trackerResponseByte.Body)

	fmt.Printf("%v\n", string(trackerResponse))

	trackerDictResponse := parser.UnmarshallFromArray(trackerResponse)

	requestErr , isPresent := trackerDictResponse.Strings["failure reason"]

	if isPresent {
		log.Fatalf("error: %v", errors.New(requestErr))
	}

	return nil, nil

}


func (peerSwarm *PeerSwarm) sendUDPTrackerRequest(event int) (*PeerProtocol.UdpMSG, error) {

	/*
	// TODO this needs to be moved
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

	 */
	return nil, nil
}

func (peerSwarm *PeerSwarm) createPeersFromHTTPTracker(peersInfo []*parser.BMap) []*Peer {
	var peers []*Peer
	for _, peerDict := range peersInfo {
		peer := peerSwarm.newPeerFromStrings(peerDict.Strings["ip"], peerDict.Strings["port"], peerDict.Strings["peer id"])
		peers = append(peers,peer)
		go peerSwarm.connect(peer)
	}
	return peers
}

func (peerSwarm *PeerSwarm) createPeersFromUDPTracker(peersAddress []byte) []*Peer {
	var peers []*Peer
	i := 0
	for i < len(peersAddress) {
		ip, port, id := peerSwarm.newPeerFromBytes(peersAddress[i:(i + 6)])
		peer := peerSwarm.newPeerFromStrings(ip, port, id)

		peers = append(peers,peer)
		///go peerSwarm.connect(peer)
		i += 6
	}

	return peers
}



func (peerSwarm *PeerSwarm) runTracker( trackerType int, ) {
	println("sending to tracker ................")

	for  {
		select {
		case <- peerSwarm.initialTrackerRequest:
		case  <- peerSwarm.trackerRequestChan.C:
			var peers []*Peer
			if trackerType == httpTracker{
				trackerResponse, err := peerSwarm.sendHTTPTrackerRequest(1, 0,0,0,"","")

				if err != nil{
					log.Fatal(err)
				}

				peers = peerSwarm.createPeersFromHTTPTracker(trackerResponse.BLists["peers"].BMaps)
			}else{

				trackerResponse, err := peerSwarm.sendUDPTrackerRequest(1)
				peers = peerSwarm.createPeersFromUDPTracker(trackerResponse.PeersAddresses)
				if err != nil{
					log.Fatal(err)
				}

			}

			// add those peers to swarm
			for _, peer := range peers{
				operation := addPeerOperation{
					peer: peer,
					swarm: peerSwarm,
				}

				peerSwarm.peerOperation <- operation
			}
		case <- peerSwarm.stopTrackerRequest:
			peerSwarm.trackerRequestChan.Stop()
			return
		}
	}

}

func (peerSwarm *PeerSwarm) stopServer()  {
	if peerSwarm.server != nil{
		err := peerSwarm.server.Close()
		if err != nil {
			// TODO NEED PROPER ERROR HANDLING
			return
		}
	}
}

func (peerSwarm *PeerSwarm) peersManager() {

	for {
		operation := <-peerSwarm.peerOperation

		operation.execute()

		/*switch operation.operation {
		case AddPeer:
			peerSwarm.addPeer(operation.peer)
		case ConnectToPeer:
			if peerSwarm.nActiveConnection < peerSwarm.maxConnection {
				go peerSwarm.connect(operation.peer)
			}else {
				//TODO should add the peer to a queue or something .......
			}

		case IncomingConnection:
			if peerSwarm.nActiveConnection < peerSwarm.maxConnection {
				go peerSwarm.handleNewPeer(operation.incomingPeerConnection)
			}
		case RemovePeer:
			peerSwarm.DropConnection(operation.peer)
		case SortPeerByDownloadRate:
			//peerSwarm.SortPeerByDownloadRate()
		case AddActivePeer:
			//peerSwarm.addActivePeer(operation.peer)
		case isPeerFree:
			var availablePeer *Peer

			for _, peer := range peerSwarm.PeerSorter.activePeers {
				if peer != nil {
					if peer.isPeerFree() {
						availablePeer = peer
						break
					}
				}
			}

			println("requesting tracker ...........")
			if availablePeer == nil && len(peerSwarm.PeerSorter.activePeers) < 2 {
				peerSwarm.torrent.LifeCycleChannel <- sendTrackerRequest
			}

			operation.freePeerChannel <- availablePeer
		case startReceivingIncomingConnection:
			if !peerSwarm.receivingIncomingConnection {
				peerSwarm.receivingIncomingConnection = true
				go peerSwarm.startServer()
			}
		case stopReceivingIncomingConnection:
			if peerSwarm.receivingIncomingConnection {
				err := peerSwarm.server.Close()
				if err != nil {
					return 
				}
			}
		}*/
	}

}









