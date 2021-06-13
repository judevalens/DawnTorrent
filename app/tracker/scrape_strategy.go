package tracker

import (
	"DawnTorrent/interfaces"
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const udpTryOutDeadline = time.Second * 15
const maxPacketSize = 65535

type scrapeStrategy interface {
	handleRequest() (int, error)
}

type httpTracker2 struct {
	*Announcer
}

func (trp *httpTracker2) execRequest() (*parser.BMap, error) {
	myID := utils.MyID

	event := trp.getCurrentState()

	//fmt.Printf("%x\n", InfoHash)

	uploaded, downloaded, left := trp.getTransferStats()

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", trp.infoHash)
	trackerRequestParam.Add("peer_id", myID)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(downloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	trackerUrl := trp.mainTrackerUrl
	trackerUrl.RawQuery = trackerRequestParam.Encode()

	fmt.Printf("\n Param \n %v \n", trackerUrl)

	trackerResponseByte, _ := http.Get(trackerUrl.String())
	trackerResponse, _ := ioutil.ReadAll(trackerResponseByte.Body)


	fmt.Printf("%v\n", string(trackerResponse))

	trackerDictResponse, _ := parser.UnmarshallFromArray(trackerResponse)

	requestErr, isPresent := trackerDictResponse.Strings["failure reason"]

	if isPresent {
		log.Fatalf("error: %v", errors.New(requestErr))
	}

	return nil, nil
}

func (trp *httpTracker2) handleRequest() (int, error) {
	print("hello from http handle request")
	var peers []interfaces.PeerI
	trackerResponse, err := trp.execRequest()
	if err != nil {
		return 0, nil
	}

	peers = trp.peerManager.CreatePeersFromHTTPTracker(trackerResponse.BLists["peers"].BMaps)

	trp.peerManager.AddNewPeer(peers...)

	return 0, nil
}


type udpTracker2 struct {
	*Announcer
}

func (trp *udpTracker2) handleRequest() (int, error) {
	var peers []interfaces.PeerI
	var err error
	connectionResponse, conn, err := trp.sendConnectRequest(time.Now().Add(udpTryOutDeadline))

	if err != nil{
		//TODO must be fixed
		log.Print(err)
		return 0, err
	}

	announceResponse,err := trp.sendAnnounceRequest(time.Now().Add(udpTryOutDeadline),conn,connectionResponse)

	if err != nil {
		log.Print(err)
		return 0, err
	}

	peers = trp.peerManager.CreatePeersFromUDPTracker(announceResponse.PeersAddresses)
	trp.Announcer.peerManager.AddNewPeer(peers...)


	//TODO need to move that in peer manager
	/*
	for _, peer := range peers {
		trp.Announcer.torrentManager.PeerManager.PeerOperationReceiver <- AddPeerOperation{
			Peer:        peer,
			PeerManager: trp.Announcer.torrentManager.PeerManager,
			MsgReceiver: trp.Announcer.torrentManager.MsgChan,
		}
	}*/

	log.Printf("announce response : interval %v, nSeeders %v, nLeechers %v", announceResponse.Interval, announceResponse.NSeeders, announceResponse.NLeechers)

	return announceResponse.Interval, nil
}

func (trp *udpTracker2) sendConnectRequest(deadline time.Time) (BaseUdpMsg, *net.UDPConn, error) {
	var err error
	var conn *net.UDPConn
	var trackerAddress *net.UDPAddr

	trackerAddress, err = net.ResolveUDPAddr("udp", trp.mainTrackerUrl.Hostname()+":"+trp.mainTrackerUrl.Port())
	if err != nil{
		log.Fatal(err)
	}
	conn, err = net.DialUDP("udp", nil, trackerAddress)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("local addr: %v,",conn.LocalAddr().String())
	randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
	transactionId := int(randSeed.Int31())
	connectionRequest := NewUdpConnectionRequest(transactionId)
	log.Printf("connectionRequest : %v",connectionRequest)
	for time.Now().UnixNano() < deadline.UnixNano() {

		log.Printf("sending udp connection request to udp tracker......")
		nByteSent, err := conn.Write(connectionRequest)
		log.Printf("n byte sent : %v",nByteSent)

		if err != nil {
			log.Fatal(err)
		}
		incomingMsg := make([]byte, maxPacketSize)
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * 15))
		nByteRead, err := bufio.NewReader(conn).Read(incomingMsg)
		log.Printf("n byte read : %v",nByteRead)

		if err != nil {

			if errors.Is(err, os.ErrDeadlineExceeded) {
				log.Print(err)
				continue
			}
			log.Fatal(err)
		}


		response := ParseUdpConnectionResponse(incomingMsg)

		log.Printf("action %v, connectionID %v, transction %v old transation %v",response.Action,response.ConnectionID,response.TransactionID,transactionId)

		//log.Printf("udp conn res:\n%v",string(incomingMsg))

		return response, conn, nil
	}
	return BaseUdpMsg{}, nil, os.ErrDeadlineExceeded
}
func (trp *udpTracker2) sendAnnounceRequest(deadline time.Time, conn *net.UDPConn,connectionResponse BaseUdpMsg) (AnnounceResponseUdpMsg, error) {
	var err error
	randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

	for time.Now().UnixNano() < deadline.UnixNano() {
		_, _, left := trp.getTransferStats()
		left = 0
		transactionId := int(randSeed.Int31())
		announceRequest := NewUdpAnnounceRequest(AnnounceUdpMsg{
			BaseUdpMsg: BaseUdpMsg{
				Action:        UdpAnnounceRequest,
				ConnectionID:  connectionResponse.ConnectionID,
				TransactionID: transactionId,
			},
			Infohash:   trp.infoHash,
			PeerId:     utils.MyID,
			Downloaded: 67,
			Uploaded:   20,
			Left:       left,
			Ip:         0,
			Event:      0,
			Key:        2,
			NumWant:    -1 , //default
			Port:       utils.LocalAddr.Port,
		})

		log.Printf("raw announce request\n%v", announceRequest)
		log.Printf("torrent state %v", trp.getCurrentStateInt())
		log.Printf("my id %v", len(utils.MyID))

		_, err = conn.Write(announceRequest)
		if err != nil {
			log.Fatal(err)
			return AnnounceResponseUdpMsg{}, err
		}

		incomingMsg := make([]byte, 1024)
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * 20))
		nByteRead, err := bufio.NewReader(conn).Read(incomingMsg)
		if err != nil {

			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			log.Fatal(err)
		}

		announceResponse := ParseAnnounceResponseUdpMsg(incomingMsg,nByteRead)

		log.Printf("action %v, connectionID %v, transction %v old transation %v",announceResponse.Action,connectionResponse.ConnectionID,announceResponse.TransactionID,transactionId)

		return announceResponse, nil
	}
	return AnnounceResponseUdpMsg{}, os.ErrDeadlineExceeded
}

