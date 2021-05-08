package protocol

import (
	_ "DawnTorrent/parser"
	"DawnTorrent/utils"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	_ "net"
	"os"
	_ "os"
	"reflect"
	"strconv"
	_ "strconv"
)

type peerManager struct {
	activePeers map[string]*Peer
	Peers       []*Peer
	PeersMap    map[string]*Peer

	nActiveConnection     int
	maxConnection         int
	peerOperationReceiver chan PeerOperation
	msgReceiver           chan BaseMsg
	server                *net.TCPListener
	InfoHashHex           string
	InfoHashByte          []byte
}

func newPeerManager(msgReceiver chan BaseMsg, infoHash string,InfoHashByte  []byte) *peerManager {
	peerManager := new(peerManager)
	peerManager.InfoHashHex = infoHash
	peerManager.InfoHashByte = InfoHashByte
	peerManager.msgReceiver = msgReceiver
	peerManager.peerOperationReceiver = make(chan PeerOperation)
	peerManager.activePeers = make(map[string]*Peer)
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

	if err != nil {
		log.Fatal(err)
	}

	// TODO no need to check the info in the parseMethod bc of separation of concern
	var handShakeMsg *HandShake

	handShakeMsg, err = parseHandShake(handShakeMsgByte)

	remotePeerAddr, _ := net.ResolveTCPAddr("tcp", connection.RemoteAddr().String())

	_, err = connection.Write(newHandShakeMsg(peerSwarm.InfoHashByte, ""))

	if err != nil {
		return
	}

	newPeer := NewPeer(remotePeerAddr.IP.String(), strconv.Itoa(remotePeerAddr.Port), handShakeMsg.peerID)
	newPeer.connection = connection

	log.Printf("incoming connection was succesful")

	peerSwarm.peerOperationReceiver <- addPeerOperation{
		peer:        newPeer,
		swarm:       peerSwarm,
		msgReceiver: peerSwarm.msgReceiver,
	}

}
func (peerSwarm *peerManager) connect(peer *Peer) error {
	var err error
	remotePeerAddr, err := net.ResolveTCPAddr("tcp", peer.ip+":"+peer.port)
	if err != nil {
		log.Fatal(err)
		return err
	}
	connection, err := net.DialTCP("tcp", nil, remotePeerAddr)
	log.Printf("connecting to: %v, conn %v\n", remotePeerAddr.String(), connection)
	if err != nil {
		log.Printf("error while connecting to peer %v :\n%v", remotePeerAddr.String(), err)
		return err
	}
	err = connection.SetKeepAlive(true)
	err = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)
	fmt.Printf("keep alive %v\n", utils.KeepAliveDuration)
	msg := newHandShakeMsg(peerSwarm.InfoHashByte, utils.MyID)

	fmt.Printf("outgoing handskahe, %v", string(msg))
	_, err = connection.Write(msg)

	if err != nil {
		log.Printf("error while sending handshake to peer %v :\n%v", remotePeerAddr.String(), err)

		return err
	}
	handshakeBytes := make([]byte, 68)
	_, err = io.ReadFull(connection, handshakeBytes)

	if err != nil {
		log.Printf("error while reading handshake from peer %v :\n%v",remotePeerAddr.String(),err)
		return err
	}

	handShakeMsg, err := parseHandShake(handshakeBytes)

	if err != nil {
		log.Fatal(err)
		os.Exit(23)
	}

	if string(handShakeMsg.infoHash) != string(peerSwarm.InfoHashByte) {
		err := connection.Close()
		if err != nil {
			log.Printf("wrong handshake from %v",remotePeerAddr.String())
			return err
		}

		log.Printf("remote hash %v, local hash %v",string(handShakeMsg.infoHash),string(peerSwarm.InfoHashByte))


		os.Exit(244)
		return errors.New("wrong infohash")
	}

	fmt.Printf("handshake, %v\n", string(handshakeBytes))

	peer.connection = connection

	return nil
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

func (peerSwarm *peerManager) startServer(ctx context.Context) {
	go func() {
		for {
			<-ctx.Done()
			peerSwarm.stopServer()
		}
	}()
	server, err := net.ListenTCP("tcp", utils.LocalAddr2)

	if err != nil {
		log.Fatalf("err: %v", err)
	}

	peerSwarm.server = server
	log.Printf("Listening on %v\n", server.Addr().String())

	for {
		var connection *net.TCPConn
		var connErr error
		connection, connErr = server.AcceptTCP()
		log.Printf("received new conn : %v\n", connection.RemoteAddr().String())
		if connErr == nil {
			print("hello")
			peerSwarm.peerOperationReceiver <- IncomingPeerConnection{
				swarm: peerSwarm,
				conn:  connection,
			}

		} else {
			log.Printf("new err: %v", err)
			return

		}

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

func (peerSwarm *peerManager) receiveOperation(ctx context.Context) {
	log.Printf("starting peer operation receiver")
	for {

		select {
		case <-ctx.Done():
			log.Printf("stopping peer operation receiver")
			return
		case operation := <-peerSwarm.peerOperationReceiver:
			log.Printf("new operation received: %v", reflect.TypeOf(operation))
			operation.execute(ctx)
		}
	}

}
