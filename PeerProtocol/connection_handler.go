package PeerProtocol

import (
	"fmt"
	"log"
	"net"
	"sync"
	"DawnTorrent/utils"
)
// TODO
// this file is ought to manage how many active DawnTorrent we can have
// but for now lets move the Listen method to peerProtocol.go
type peerConnectionHandler struct {
	Torrent
	maxConnection	int
	activeConnection	*[]Peer
	nActiveConnection int
	peerConnectionBlocker 		sync.WaitGroup

}


func (peerConnectionHandler *peerConnectionHandler) Listen() {
	sever, err := net.ListenTCP("tcp", utils.LocalAddr2)
	fmt.Printf("Listenning on %v\n", sever.Addr().String())

	for {
		if err == nil {
			connection, connErr := sever.AcceptTCP()
			if connErr == nil {

				if peerConnectionHandler.maxConnection < peerConnectionHandler.nActiveConnection {
					println("newPeer\n")

					//go DawnTorrent.handleNewPeer(connection)
				}
			} else {
				_ = connection.Close()
			}
		} else {
			println("eer")
			log.Fatal(err)
		}

	}
}