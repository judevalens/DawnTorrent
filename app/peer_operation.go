package app

import (
	"context"
	"net"
)



type PeerOperation interface {
	execute(ctx context.Context)
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
	err := operation.swarm.connect(operation.peer)
	if err != nil {
		return
	}
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


