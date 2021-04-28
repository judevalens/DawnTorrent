package protocol

import (
	_ "DawnTorrent/parser"
	"DawnTorrent/utils"
	"fmt"
	"io"
	"log"
	"net"
	_ "net"
	_ "os"
	"strconv"
	_ "strconv"
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



type peerManager struct {
	torrentManager *TorrentManager
	activePeers                 map[string]*Peer
	Peers                       []*Peer
	PeersMap                    map[string]*Peer
	interestedPeerIndex         []int
	interestedPeerMap           map[string]*Peer
	interestingPeer             map[string]*Peer
	unChockedPeer               []*Peer
	unChockedPeerMap            map[string]*Peer

	nActiveConnection           int
	maxConnection               int
	torrent                     *Torrent
	trackerInterval             int
	peerOperationReceiver       chan PeerOperation
	lastSelectedPeer            int
	trackerRequestChan          time.Ticker
	initialTrackerRequest       chan interface{}
	stopTrackerRequest          chan interface{}
	receivingIncomingConnection bool
	server                      *net.TCPListener
}

func newPeerManager() peerManager{
	peerManager := peerManager{}

	return peerManager
}

func (peerSwarm *peerManager) addPeer(peer *Peer) *Peer {
	peerSwarm.PeersMap[peer.id] = peer
	// TODO That's probably redundant
	peerSwarm.Peers = append(peerSwarm.Peers, peer)
	peer.peerIndex = len(peerSwarm.Peers) - 1
	return peer
}
func (peerSwarm *peerManager) handleNewPeer(connection *net.TCPConn) {
	//var newPeer *Peer
	var err error
	_ = connection.SetKeepAlive(true)
	_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)

	handShakeMsgByte := make([]byte, 68)
	_, err = io.ReadFull(connection, handShakeMsgByte)

	if err != nil{
		log.Fatal(err)
	}

		// TODO no need to check the info in the parseMethod bc of separation of concern
		var handShakeMsg *HandShake

		handShakeMsg, err = parseHandShake(handShakeMsgByte)


				remotePeerAddr, _ := net.ResolveTCPAddr("tcp", connection.RemoteAddr().String())

				_, err = connection.Write(newHandShakeMsg(peerSwarm.torrent.InfoHashHex, ""))

				if err == nil {

					newPeer := NewPeer(remotePeerAddr.IP.String(), strconv.Itoa(remotePeerAddr.Port), handShakeMsg.peerID)
					newPeer.connection = connection

					peerSwarm.peerOperationReceiver <- addPeerOperation{
						peer: newPeer,
						swarm: peerSwarm,
					}

				}

}

func (peerSwarm *peerManager) connect(peer *Peer) {
	var err error
	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	connection, err := net.DialTCP("tcp", nil, remotePeerAddr)
	fmt.Printf("peer addr : %v, con %v\n", remotePeerAddr.String(), connection)
	if err == nil {
		_ = connection.SetKeepAlive(true)
		_ = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)
		fmt.Printf("keep ALive %v", utils.KeepAliveDuration)
		_, err := connection.Write(newHandShakeMsg(peerSwarm.torrent.InfoHashHex, ""))

		if err != nil{
			log.Fatal(err)
		}
		handshakeBytes := make([]byte, 68)
		_, err = io.ReadFull(connection, handshakeBytes)

			if err == nil {
				handShakeMsg, handShakeMsgErr := parseHandShake(handshakeBytes)

				if handShakeMsgErr != nil{
					log.Fatal(handShakeMsgErr)
				}

				if handShakeMsg.infoHash != peerSwarm.torrent.InfoHashHex{
					return
				}

					peer.connection = connection
					peerSwarm.peerOperationReceiver <- addPeerOperation{
						peer:  peer,
						swarm: peerSwarm,
					}
			}
	} else {
		//fmt.Printf("connection failed \n")
	}
}
func (peerSwarm *peerManager) DropConnection(peer *Peer) {
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

func (peerSwarm *peerManager) startServer() {
	server, err := net.ListenTCP("tcp", utils.LocalAddr2)
	peerSwarm.server = server
	//fmt.Printf("Listening on %v\n", server.Addr().String())
	if err == nil {
		for {
			var connection *net.TCPConn
			var connErr error
			connection, connErr = server.AcceptTCP()
			if connErr == nil {

				peerSwarm.peerOperationReceiver <- IncomingPeerConnection{
					swarm: peerSwarm,
					conn:  connection,
				}

			} else {
				log.Printf("new err: %v", err)
				return

			}

		}

	} else {
		log.Fatalf("error: %v", err)
	}
}

func (peerSwarm *peerManager) stopServer() {
	if peerSwarm.server != nil {
		err := peerSwarm.server.Close()
		if err != nil {
			// TODO NEED PROPER ERROR HANDLING
			return
		}
	}
}

func (peerSwarm *peerManager) receiveOperation() {

	for {
		operation := <-peerSwarm.peerOperationReceiver

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

			println("requesting baseTracker ...........")
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
