package protocol

import "net"



type PeerOperation2 interface {
	execute()
}

type addPeerOperation struct {
	peer *Peer
	swarm *PeerSwarm
}

func(operation addPeerOperation) execute(){
	operation.swarm.addPeer(operation.peer)
}

type dropPeerOperation struct {
	peer *Peer
	swarm *PeerSwarm
}

func (operation dropPeerOperation) execute() {
	operation.swarm.DropConnection(operation.peer)
}

type IncommingPeerConnection struct {
	conn *net.TCPConn
	swarm *PeerSwarm
}

func (operation IncommingPeerConnection) execute() {
	operation.swarm.handleNewPeer(operation.conn)
}

type connectPeerOperation struct {
	peer *Peer
	swarm *PeerSwarm
}

func (operation connectPeerOperation) execute() {
	operation.swarm.connect(operation.peer)
}

type startServer struct {
	swarm *PeerSwarm
}

func (operation connectPeerOperation) receiveIncomingConnection() {
	operation.swarm.startServer()
}

type stopSever struct {
	swarm *PeerSwarm
}

func (operation connectPeerOperation) stopIncomingConnection() {
	operation.swarm.stopServer()
}


