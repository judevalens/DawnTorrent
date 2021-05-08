package app

import (
	"DawnTorrent/parser"
	_ "DawnTorrent/parser"
	"DawnTorrent/protocol"
	"DawnTorrent/utils"
	"bytes"
	"context"
	"encoding/binary"
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
	PeerOperationReceiver chan PeerOperation
	msgReceiver           chan torrentMsg
	server                *net.TCPListener
	InfoHashHex           string
	InfoHashByte          []byte
}

func (manager *peerManager) HandleMsgStream(ctx context.Context, peer protocol.PeerI) error{
	var err error
	msgLenBuffer := make([]byte, 4)
	for  {

		//reads the length of the incoming msg
		_, err = io.ReadFull(peer.GetConnection(), msgLenBuffer)

		msgLen := int(binary.BigEndian.Uint32(msgLenBuffer[0:4]))

		// reads the full payload
		incomingMsgBuffer := make([]byte, msgLen)
		_, err = io.ReadFull(peer.GetConnection(), incomingMsgBuffer)
		if err != nil{
			log.Fatal(err)
		}
		msg, err := ParseMsg(bytes.Join([][]byte{msgLenBuffer, incomingMsgBuffer}, []byte{}), peer)

		log.Printf("received new msg from : %v, \n %v",peer.GetConnection().RemoteAddr().String(), msg)

		if err != nil {
			os.Exit(23)
		}
		manager.msgReceiver <- msg

	}
}

func (manager *peerManager) GetActivePeers() map[string]protocol.PeerI {
	panic("implement me")
}

func (manager *peerManager) AddNewPeer(peers []protocol.PeerI) {

	for _, peer := range peers {
		currentPeer := peer

		go func() {
			if currentPeer.GetConnection() == nil {
				err := manager.connect(currentPeer)
				if err != nil {
					return
				}
			}
			err := manager.HandleMsgStream(nil,currentPeer)
			if err != nil {
				log.Printf("something bad happen while peer was connected\n err: %v", err)
			}
		}()
	}

}

func newPeerManager(msgReceiver chan torrentMsg, infoHash string, InfoHashByte []byte) *peerManager {
	peerManager := new(peerManager)
	peerManager.InfoHashHex = infoHash
	peerManager.InfoHashByte = InfoHashByte
	peerManager.msgReceiver = msgReceiver
	peerManager.PeerOperationReceiver = make(chan PeerOperation)
	peerManager.activePeers = make(map[string]*Peer)
	return peerManager
}

func (manager *peerManager) addPeer(peer *Peer) *Peer {
	manager.PeersMap[peer.id] = peer
	// TODO That's probably redundant
	manager.Peers = append(manager.Peers, peer)
	peer.peerIndex = len(manager.Peers) - 1
	return peer
}
func (manager *peerManager) handleConnectionRequest(connection *net.TCPConn) {
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

	_, err = connection.Write(newHandShakeMsg(manager.InfoHashByte, ""))

	if err != nil {
		return
	}

	newPeer := NewPeer(remotePeerAddr.IP.String(), strconv.Itoa(remotePeerAddr.Port), handShakeMsg.peerID)
	newPeer.connection = connection

	log.Printf("incoming connection was succesful")

	manager.AddNewPeer([]protocol.PeerI{newPeer})

}
func (manager *peerManager) connect(peer protocol.PeerI) error {
	var err error
	if err != nil {
		log.Fatal(err)
		return err
	}
	connection, err := net.DialTCP("tcp", nil, peer.GetAddress())
	log.Printf("connecting to: %v, conn %v\n", peer.GetAddress().String(), connection)
	if err != nil {
		log.Printf("error while connecting to peer %v :\n%v", peer.GetAddress().String(), err)
		return err
	}
	err = connection.SetKeepAlive(true)
	err = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)
	fmt.Printf("keep alive %v\n", utils.KeepAliveDuration)
	msg := newHandShakeMsg(manager.InfoHashByte, utils.MyID)

	fmt.Printf("outgoing handskahe, %v", string(msg))
	_, err = connection.Write(msg)

	if err != nil {
		log.Printf("error while sending handshake to peer %v :\n%v", peer.GetAddress().String(), err)

		return err
	}
	handshakeBytes := make([]byte, 68)
	_, err = io.ReadFull(connection, handshakeBytes)

	if err != nil {
		log.Printf("error while reading handshake from peer %v :\n%v", peer.GetAddress().String(), err)
		return err
	}

	handShakeMsg, err := parseHandShake(handshakeBytes)

	if err != nil {
		log.Fatal(err)
		os.Exit(23)
	}

	if string(handShakeMsg.infoHash) != string(manager.InfoHashByte) {
		err := connection.Close()
		if err != nil {
			log.Printf("wrong handshake from %v", peer.GetAddress().String())
			return err
		}

		log.Printf("remote hash %v, local hash %v", string(handShakeMsg.infoHash), string(manager.InfoHashByte))

		os.Exit(244)
		return errors.New("wrong infohash")
	}

	fmt.Printf("handshake, %v\n", string(handshakeBytes))

	peer.SetConnection(connection)

	return nil
}

func (manager *peerManager) CreatePeersFromUDPTracker(peersAddress []byte) []protocol.PeerI {
	var peers []protocol.PeerI
	i := 0
	log.Printf("peersAddress len %v",len(peersAddress))
	for i < len(peersAddress) {
		peer := NewPeerFromBytes(peersAddress[i:(i + 6)])
		peers = append(peers, peer)
		///go peerSwarm.connect(peer)
		i += 6
	}

	return peers
}

func (manager *peerManager) CreatePeersFromHTTPTracker(peersInfo []*parser.BMap) []protocol.PeerI {
	var peers []protocol.PeerI
	for _, peerDict := range peersInfo {
		peer := NewPeer(peerDict.Strings["ip"], peerDict.Strings["port"], peerDict.Strings["peer id"])
		peers = append(peers, peer)

	}
	return peers
}

func (manager *peerManager) DropConnection(peer *Peer) {
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

func (manager *peerManager) startServer(ctx context.Context) {
	go func() {
		for {
			<-ctx.Done()
			manager.stopServer()
		}
	}()
	server, err := net.ListenTCP("tcp", utils.LocalAddr2)

	if err != nil {
		log.Fatalf("err: %v", err)
	}

	manager.server = server
	log.Printf("Listening on %v\n", server.Addr().String())

	for {
		var connection *net.TCPConn
		var connErr error
		connection, connErr = server.AcceptTCP()
		log.Printf("received new conn : %v\n", connection.RemoteAddr().String())
		if connErr == nil {
			print("hello")
			manager.PeerOperationReceiver <- IncomingPeerConnection{
				swarm: manager,
				conn:  connection,
			}

		} else {
			log.Printf("new err: %v", err)
			return

		}

	}

}

func (manager *peerManager) stopServer() {
	if manager.server != nil {
		err := manager.server.Close()
		if err != nil {
			// TODO NEED PROPER ERROR HANDLING
			return
		}
	}
}

func (manager *peerManager) receiveOperation(ctx context.Context) {
	log.Printf("starting peer operation receiver")
	for {

		select {
		case <-ctx.Done():
			log.Printf("stopping peer operation receiver")
			return
		case operation := <-manager.PeerOperationReceiver:
			log.Printf("new operation received: %v", reflect.TypeOf(operation))
			operation.execute(ctx)
		}
	}

}
