package app

import (
	"context"
	"net"
)





type dropPeerOperation struct {
	peer *Peer
	swarm *peerManager
}

func (operation dropPeerOperation) Execute(context.Context) {
	operation.swarm.DropPeer(operation.peer)
}

type IncomingPeerConnection struct {
	conn *net.TCPConn
	swarm *peerManager
}

func (operation IncomingPeerConnection) Execute(context.Context) {
	operation.swarm.handleConnectionRequest(operation.conn)
}

type connectPeerOperation struct {
	peer *Peer
	swarm *peerManager
}

func (operation connectPeerOperation) Execute(context.Context) {
	err := operation.swarm.connect(operation.peer)
	if err != nil {
		return
	}
}

type startServer struct {
	swarm *peerManager
}

func (operation startServer) Execute(ctx context.Context) {
	go operation.swarm.startServer(ctx)
}

type stopServer struct {
	swarm *peerManager
}

func (operation stopServer) Execute(context.Context) {
	operation.swarm.stopServer()
}


