package app

import (
	"DawnTorrent/interfaces"
	"DawnTorrent/parser"
	_ "DawnTorrent/parser"
	"DawnTorrent/utils"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	_ "net"
	"os"
	_ "os"
	"reflect"
	"strconv"
	_ "strconv"
)

type PeerManager struct {
	activePeers map[string]*Peer
	peersId []string

	nActiveConnection     int
	maxConnection         int
	PeerOperationReceiver chan interfaces.Operation
	msgReceiver           chan TorrentMsg
	server                *net.TCPListener
	InfoHashHex           string
	InfoHashByte          []byte
}

func newPeerManager(msgReceiver chan TorrentMsg, infoHash string, InfoHashByte []byte) *PeerManager {
	peerManager := new(PeerManager)
	peerManager.InfoHashHex = infoHash
	peerManager.InfoHashByte = InfoHashByte
	peerManager.msgReceiver = msgReceiver
	peerManager.PeerOperationReceiver = make(chan interfaces.Operation)
	peerManager.activePeers = make(map[string]*Peer)
	return peerManager
}



func (manager *PeerManager) HandleMsgStream(ctx context.Context, peer *Peer) error{
	log.Infof("connection established to: %v",peer.id)
	var err error
	msgLenBuffer := make([]byte, 4)
	for  {

		//reads the length of the incoming msg
		_, err = io.ReadFull(peer.GetConnection(), msgLenBuffer)

		if err != nil{
			log.Fatal(err)
			continue
		}

		msgLen := int(binary.BigEndian.Uint32(msgLenBuffer[0:4]))

		// reads the full payload
		incomingMsgBuffer := make([]byte, msgLen)
		_, err = io.ReadFull(peer.GetConnection(), incomingMsgBuffer)
		if err != nil{
			log.Fatal(err)
		}
		msg, err := ParseMsg(bytes.Join([][]byte{msgLenBuffer, incomingMsgBuffer}, []byte{}), peer)

		if err != nil{
			log.Errorf("incorrect msg:\n,%v", bytes.Join([][]byte{msgLenBuffer, incomingMsgBuffer}, []byte{}))
			log.Fatal(err)
		}

		log.Infof("received new msg from : %v, \n %v",peer.id, msg.getId())

		if err != nil {
			os.Exit(23)
		}
		manager.msgReceiver <- msg

	}
}

func (manager *PeerManager) GetActivePeers() map[string]interfaces.PeerI {
	panic("implement me")
}

func (manager *PeerManager) AddNewPeer(peers ...interfaces.PeerI) {

	for _, peer := range peers {
		go func(peer *Peer) {
			if peer.GetConnection() == nil {
				err := manager.connect(peer)
				if err != nil {
					return
				}
			}
			manager.activePeers[peer.GetId()] = peer

			err := manager.HandleMsgStream(nil,peer)

			if err != nil {
				log.Printf("something bad happen while peer was connected\n err: %v", err)
			}
		}(peer.(*Peer))
	}

}


func (manager *PeerManager) GetAvailablePeer(reqId string, pieceIndex int) (*Peer, error) {
	for _, peer := range manager.activePeers {
		if peer.isAvailable(reqId) && peer.HasPiece(pieceIndex) {
			return peer,nil
		}
	}
	return nil, errors.New("no peers available")
}

func (manager *PeerManager) handleConnectionRequest(connection *net.TCPConn) {
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
	manager.AddNewPeer(newPeer)

}

func (manager *PeerManager) connect(peer interfaces.PeerI) error {
	var err error
	if err != nil {
		log.Fatal(err)
		return err
	}
	connection, err := net.DialTCP("tcp", nil, peer.GetAddress())
	log.Infof("connecting to: %v, conn %v\n", peer.GetAddress().String(), connection)
	if err != nil {
		log.Error("error while connecting to peer %v :\n%v", peer.GetAddress().String(), err)
		return err
	}
	err = connection.SetKeepAlive(true)
	err = connection.SetKeepAlivePeriod(utils.KeepAliveDuration)
	msg := newHandShakeMsg(manager.InfoHashByte, utils.MyID)

	_, err = connection.Write(msg)

	if err != nil {
		log.Errorf("error while sending handshake to peer %v :\n%v", peer.GetAddress().String(), err)
		os.Exit(23)
		return err
	}
	handshakeBytes := make([]byte, 68)
	_, err = io.ReadFull(connection, handshakeBytes)

	if err != nil {
		log.Errorf("error while reading handshake from peer %v :\n%v", peer.GetAddress().String(), err)
		return err
	}

	handShakeMsg, err := parseHandShake(handshakeBytes)

	if err != nil {
		log.Fatal(err)
		os.Exit(233)
	}

	if string(handShakeMsg.infoHash) != string(manager.InfoHashByte) {
		err := connection.Close()
		if err != nil {
			log.Printf("wrong handshake from %v", peer.GetAddress().String())
			return err
		}

		log.Debug("remote hash %v, local hash %v", string(handShakeMsg.infoHash), string(manager.InfoHashByte))

		os.Exit(244)
		return errors.New("wrong infohash")
	}

	log.Debug("handshake, %v\n", string(handshakeBytes))

	peer.SetConnection(connection)
	return nil
}

func (manager *PeerManager) CreatePeersFromUDPTracker(peersAddress []byte) []interfaces.PeerI {
	var peers []interfaces.PeerI
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

func (manager *PeerManager) CreatePeersFromHTTPTracker(peersInfo []*parser.BMap) []interfaces.PeerI {
	var peers []interfaces.PeerI
	for _, peerDict := range peersInfo {
		peer := NewPeer(peerDict.Strings["ip"], peerDict.Strings["port"], peerDict.Strings["peer id"])
		peers = append(peers, peer)

	}
	return peers
}

func (manager *PeerManager) DropPeer(peer *Peer) {
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

func (manager *PeerManager) startServer(ctx context.Context) {
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

func (manager *PeerManager) stopServer() {
	if manager.server != nil {
		err := manager.server.Close()
		if err != nil {
			// TODO NEED PROPER ERROR HANDLING
			return
		}
	}
}

func (manager *PeerManager) receiveOperation(ctx context.Context) {
	log.Printf("starting peer operation receiver")
	for {

		select {
		case <-ctx.Done():
			log.Printf("stopping peer operation receiver")
			return
		case operation := <-manager.PeerOperationReceiver:
			log.Printf("new operation received: %v", reflect.TypeOf(operation))
			operation.Execute(ctx)
		}
	}

}

func (manager *PeerManager) updateInterest()  {
	for _, peer := range manager.activePeers {
		if peer.interestPoint == 0 && peer.isInteresting{
			_, err := peer.SendMsg(InterestedMsg{
				header{
					ID:     UnInterestedMsgId,
					Length: defaultMsgLen,
				},
			}.Marshal())
			if err != nil {
				log.Error(err)
				continue
			}
			peer.isInteresting = false
		}
	}
}