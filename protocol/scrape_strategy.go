package protocol

import (
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
	*Scrapper
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
	var peers []*Peer
	trackerResponse, err := trp.execRequest()
	if err != nil {
		return 0, nil
	}

	peers = trp.createPeersFromHTTPTracker(trackerResponse.BLists["peers"].BMaps)

	for _, peer := range peers {
		operation := addPeerOperation{
			peer:  peer,
			swarm: trp.Scrapper.manager.peerManager,
		}
		trp.Scrapper.manager.peerManager.peerOperationReceiver <- operation
	}

	return 0, nil
}
func (trp *httpTracker2) createPeersFromHTTPTracker(peersInfo []*parser.BMap) []*Peer {
	var peers []*Peer
	for _, peerDict := range peersInfo {
		peer := NewPeer(peerDict.Strings["ip"], peerDict.Strings["port"], peerDict.Strings["peer id"])
		peers = append(peers, peer)
		//go peerSwarm.connect(peer)
	}
	return peers
}

type udpTracker2 struct {
	*Scrapper
}

func (trp *udpTracker2) handleRequest() (int, error) {
	var peers []*Peer
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

	peers = trp.createPeersFromUDPTracker(announceResponse.peersAddresses)

	for _, peer := range peers {

		trp.Scrapper.manager.peerManager.peerOperationReceiver <- addPeerOperation{
			peer:  peer,
			swarm: trp.Scrapper.manager.peerManager,
		}
	}

	log.Printf("announce response : interval %v, nSeeders %v, nLeechers %v", announceResponse.interval, announceResponse.nSeeders, announceResponse.nLeechers)

	return announceResponse.interval, nil
}

func (trp *udpTracker2) sendConnectRequest(deadline time.Time) (baseUdpMsg, *net.UDPConn, error) {
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
	connectionRequest := newUdpConnectionRequest(transactionId)
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


		response := parseUdpConnectionResponse(incomingMsg)

		log.Printf("action %v, connectionID %v, transction %v old transation %v",response.action,response.connectionID,response.transactionID,transactionId)

		//log.Printf("udp conn res:\n%v",string(incomingMsg))

		return response, conn, nil
	}
	return baseUdpMsg{}, nil, os.ErrDeadlineExceeded
}
func (trp *udpTracker2) sendAnnounceRequest(deadline time.Time, conn *net.UDPConn,connectionResponse baseUdpMsg) (announceResponseUdpMsg, error) {
	var err error
	randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

	for time.Now().UnixNano() < deadline.UnixNano() {
		_, _, left := trp.getTransferStats()
		left = 0
		transactionId := int(randSeed.Int31())
		announceRequest := newUdpAnnounceRequest(announceUdpMsg{
			baseUdpMsg: baseUdpMsg{
				action:        udpAnnounceRequest,
				connectionID:  connectionResponse.connectionID,
				transactionID: transactionId,
			},
			infohash:   trp.infoHash,
			peerId:     utils.MyID,
			downloaded: 67,
			uploaded:   20,
			left:       left,
			ip: 0,
			event:      0,
			key:        2,
			numWant: 	-1 , //default
			port: utils.LocalAddr.Port,
		})

		log.Printf("raw announce request\n%v", announceRequest)
		log.Printf("torrent state %v", trp.getCurrentStateInt())
		log.Printf("my id %v", len(utils.MyID))

		_, err = conn.Write(announceRequest)
		if err != nil {
			log.Fatal(err)
			return announceResponseUdpMsg{}, err
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

		announceResponse := parseAnnounceResponseUdpMsg(incomingMsg,nByteRead)

		log.Printf("action %v, connectionID %v, transction %v old transation %v",announceResponse.action,connectionResponse.connectionID,announceResponse.transactionID,transactionId)

		return announceResponse, nil
	}
	return announceResponseUdpMsg{}, os.ErrDeadlineExceeded
}

func (trp *udpTracker2) createPeersFromUDPTracker(peersAddress []byte) []*Peer {
	var peers []*Peer
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

