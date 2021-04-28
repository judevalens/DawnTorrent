package protocol

import "net"



type PeerOperation interface {
	execute()
}

type addPeerOperation struct {
	peer *Peer
	swarm *peerManager
}

func(operation addPeerOperation) execute(){
 }

type dropPeerOperation struct {
	peer *Peer
	swarm *peerManager
}

func (operation dropPeerOperation) execute() {
	operation.swarm.DropConnection(operation.peer)
}

type IncomingPeerConnection struct {
	conn *net.TCPConn
	swarm *peerManager
}

func (operation IncomingPeerConnection) execute() {
	operation.swarm.handleNewPeer(operation.conn)
}

type connectPeerOperation struct {
	peer *Peer
	swarm *peerManager
}

func (operation connectPeerOperation) execute() {
	operation.swarm.connect(operation.peer)
}

type startServer struct {
	swarm *peerManager
}

func (operation startServer) execute() {
	go operation.swarm.startServer()
}

type stopServer struct {
	swarm *peerManager
}

func (operation stopServer) execute() {
	operation.swarm.stopServer()
}

