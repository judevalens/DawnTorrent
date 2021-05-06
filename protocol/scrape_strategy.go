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

const udpTryOutDeadline = time.Second * 60
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

	trackerUrl := trp.trackerURL
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
			swarm: trp.Scrapper.peerManager,
		}
		trp.Scrapper.peerManager.peerOperationReceiver <- operation
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
		log.Fatal(err)
	}

	announceResponse,err := trp.sendAnnounceRequest(time.Now().Add(udpTryOutDeadline),conn,connectionResponse)

	if err != nil {
		return 0, nil
	}

	peers = trp.createPeersFromUDPTracker(announceResponse.peersAddresses)

	for _, peer := range peers {
		operation := addPeerOperation{
			peer:  peer,
			swarm: trp.Scrapper.peerManager,
		}
		trp.Scrapper.peerManager.peerOperationReceiver <- operation
	}

	return 0, nil
}

func (trp *udpTracker2) sendConnectRequest(deadline time.Time) (baseUdpMsg, *net.UDPConn, error) {
	var err error
	var conn *net.UDPConn
	for time.Now().UnixNano() > deadline.UnixNano() {
		randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
		connectionRequest := newUdpConnectionRequest(int(randSeed.Int31()))

		cleanedUdpAddr := trp.trackerURL
		trackerAddress, _ := net.ResolveUDPAddr("udp", cleanedUdpAddr.String())

		conn, err = net.DialUDP("udp", nil, trackerAddress)
		if err != nil {
			log.Fatal(err)
		}
		_, err = conn.Write(connectionRequest)

		if err != nil {
			log.Fatal(err)
		}
		incomingMsg := make([]byte, 5000)
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * 15))
		_, err = bufio.NewReader(conn).Read(incomingMsg)
		if err != nil {

			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			log.Fatal(err)
		}

		return parseUdpConnectionResponse(incomingMsg), conn, nil
	}
	return baseUdpMsg{}, nil, os.ErrDeadlineExceeded
}
func (trp *udpTracker2) sendAnnounceRequest(deadline time.Time, conn *net.UDPConn,connectionResponse baseUdpMsg) (announceResponseUdpMsg, error) {
	var err error
	randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

	for time.Now().UnixNano() > deadline.UnixNano() {
		downloaded, uploaded, left := trp.getTransferStats()
		announceRequest := newUdpAnnounceRequest(announceUdpMsg{
			baseUdpMsg: connectionResponse,
			infohash:   trp.infoHash,
			peerId:     utils.MyID,
			downloaded: downloaded,
			uploaded:   uploaded,
			left:       left,
			event:      trp.getCurrentStateInt(),
			key:        int(randSeed.Int31()),
			numWant: 	-1 , //default
			port: utils.LocalAddr.Port,
		})

		_, err = conn.Write(announceRequest)
		if err != nil {
			return announceResponseUdpMsg{}, err
		}

		incomingMsg := make([]byte, maxPacketSize)
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * 15))
		nByteRead, err := bufio.NewReader(conn).Read(incomingMsg)
		if err != nil {

			if errors.Is(err, os.ErrDeadlineExceeded) {
				continue
			}
			log.Fatal(err)
		}

		return parseAnnounceResponseUdpMsg(incomingMsg,nByteRead), nil
	}
	return announceResponseUdpMsg{}, os.ErrDeadlineExceeded
}

func (trp *udpTracker2) createPeersFromUDPTracker(peersAddress []byte) []*Peer {
	var peers []*Peer
	i := 0
	for i < len(peersAddress) {
		peer := NewPeerFromBytes(peersAddress[i:(i + 6)])
		peers = append(peers, peer)
		///go peerSwarm.connect(peer)
		i += 6
	}

	return peers
}

type UdpMSG struct {
	action         int
	connectionID   int
	transactionID  int
	infoHash       []byte
	peerID         []byte
	downloaded     int
	left           int
	uploaded       int
	event          int
	ip             []byte
	key            int
	numWant        int
	port           int
	Interval       int
	leechers       int
	seeders        int
	PeersAddresses []byte
}
