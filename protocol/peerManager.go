package protocol

import (
	"github.com/emirpasic/gods/maps/hashmap"
	_ "net"
	"os"
	"sync"
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
	peerOperation                   chan *peerOperation
	lastSelectedPeer                int
	stopReceivingIncomingConnection chan bool
	receivingIncomingConnection     bool
}

type peerOperation struct {
	operation int
	peer      *Peer
	// used to get an available peer
	freePeerChannel chan *Peer
}


func (peerSwarm *PeerSwarm) addPeer(peer *Peer) *Peer {
	peerSwarm.PeersMap[peer.id] = peer
	// TODO That's probably redundant
	peerSwarm.Peers = append(peerSwarm.Peers, peer)
	peer.peerIndex = len(peerSwarm.Peers) - 1
	return peer
}


func (peerSwarm *PeerSwarm) peersManager() {

	for {
		var operation *peerOperation
		operation = <-peerSwarm.peerOperation

		switch operation.operation {
		case AddPeer:
			peerSwarm.addPeer(operation.peer)
		case ConnectToPeer:
			if len(peerSwarm.PeerSorter.activePeers) < peerSwarm.maxConnection {
				go peerSwarm.connect(operation.peer)
			}
		case IncomingConnection:
			if len(peerSwarm.PeerSorter.activePeers) < peerSwarm.maxConnection {
				go peerSwarm.handleNewPeer(operation.connection)
			}
		case RemovePeer:
			peerSwarm.DropConnection(operation.peer)
		case SortPeerByDownloadRate:
			peerSwarm.SortPeerByDownloadRate()
		case AddActivePeer:
			peerSwarm.addActivePeer(operation.peer)
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
