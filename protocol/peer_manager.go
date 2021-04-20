package protocol

import (
	"DawnTorrent/PeerProtocol"
	_ "DawnTorrent/parser"
	"DawnTorrent/utils"
	"fmt"
	"github.com/emirpasic/gods/maps/hashmap"
	"io"
	"log"
	"net"
	_ "net"
	_ "os"
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



type PeerSwarm struct {
	activePeers                 map[string]*Peer
	Peers                       []*Peer
	PeersMap                    map[string]*Peer
	interestedPeerIndex         []int
	interestedPeerMap           map[string]*Peer
	interestingPeer             map[string]*Peer
	unChockedPeer               []*Peer
	unChockedPeerMap            map[string]*Peer
	peerMutex                   *sync.RWMutex
	interestingPeerMutex        sync.Mutex
	activeConnectionMutex       *sync.RWMutex
	activeConnection            *hashmap.Map
	nActiveConnection           int
	maxConnection               int
	torrent                     *Torrent
	trackerInterval             int
	peerOperation               chan PeerOperation2
	lastSelectedPeer            int
	trackerRequestChan          time.Ticker
	initialTrackerRequest       chan interface{}
	stopTrackerRequest          chan interface{}
	receivingIncomingConnection bool
	server                      *net.TCPListener
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
	fmt.Printf("peer addr : %v, con %v\n", remotePeerAddr.String(), connection)
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
	peerSwarm.server = server
	//fmt.Printf("Listening on %v\n", server.Addr().String())
	if err == nil {
		for {
			var connection *net.TCPConn
			var connErr error
			connection, connErr = server.AcceptTCP()
			if connErr == nil {

				peerSwarm.peerOperation <- IncommingPeerConnection{
					swarm: peerSwarm,
					conn:  connection,
				}

			} else {
				log.Fatalf("error: %v", err)

			}

		}

	} else {
		log.Fatalf("error: %v", err)
	}
}

func (peerSwarm *PeerSwarm) stopServer() {
	if peerSwarm.server != nil {
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
