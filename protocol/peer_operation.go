package protocol

import (
	"context"
	"net"
)



type PeerOperation interface {
	execute(ctx context.Context)
}

type addPeerOperation struct {
	peer *Peer
	swarm *peerManager
	msgReceiver chan BaseMsg
}

func(operation addPeerOperation) execute(ctx context.Context) {

	// for now, we set no limit on the max connections allowed
	operation.swarm.activePeers[operation.peer.id] = operation.peer
	go func() {
		err := operation.peer.receive(ctx, operation.msgReceiver)
		if err != nil {

		}
	}()
}

type dropPeerOperation struct {
	peer *Peer
	swarm *peerManager
}

func (operation dropPeerOperation) execute(context.Context) {
	operation.swarm.DropConnection(operation.peer)
}

type IncomingPeerConnection struct {
	conn *net.TCPConn
	swarm *peerManager
}

func (operation IncomingPeerConnection) execute(context.Context) {
	operation.swarm.handleNewPeer(operation.conn)
}

type connectPeerOperation struct {
	peer *Peer
	swarm *peerManager
}

func (operation connectPeerOperation) execute(context.Context) {
	operation.swarm.connect(operation.peer)
}

type startServer struct {
	swarm *peerManager
}

func (operation startServer) execute(ctx context.Context) {
	go operation.swarm.startServer(ctx)
}

type stopServer struct {
	swarm *peerManager
}

func (operation stopServer) execute(context.Context) {
	operation.swarm.stopServer()
}


