package protocol

import (
	"DawnTorrent/PeerProtocol"
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"
)

const (
	httpTracker = iota
	udpTracker  = iota
)

const (
	httpScheme = "https"
	udpScheme  = "udp"
)

type tracker interface {
	handleRequest() (int, error)
	getCurrentState() string
	getAnnouncerUrl() *url.URL
	getInfoHash() string
	getTransferStats() (int, int, int)
	starTracker(ctx context.Context)
	stopTracker()
	resetTracker(duration time.Duration)
}

type baseTracker struct {
	state       trackerRequestState
	infoHash 	string
	trackerURL  *url.URL
	interval    time.Duration
	timer       *time.Timer
	peerManager *peerManager
}

func newTracker(announcerUrlString ,infoHash string, peerManager *peerManager) tracker {
	var tracker tracker
	trackerURL, err := url.Parse(announcerUrlString)

	if err != nil {
		return nil
	}

	baseTracker := baseTracker{
		peerManager: peerManager,
		infoHash: infoHash,
		trackerURL:  trackerURL,
		timer:       time.NewTimer(time.Nanosecond),
	}
	log.Printf("url: %v, url scheme : %v",announcerUrlString,trackerURL.Scheme )
	if trackerURL.Scheme == httpScheme {
		tracker = &httpTracker2{
			baseTracker: &baseTracker,
		}
	} else {
		tracker = &udpTracker2{
			baseTracker: &baseTracker,
		}
	}

	log.Printf("creating new tracker: %v", reflect.TypeOf(tracker))

	return tracker
}

func (t *baseTracker) handleRequest() (int, error) {
	panic("implement me")
}

func (t *baseTracker) getCurrentState() string {
	return ""
}

func (t *baseTracker) getAnnouncerUrl() *url.URL {
	return nil
}
func (t *baseTracker) getInfoHash() string {
	return ""
}

func (t *baseTracker) getTransferStats() (int, int, int) {
	return 0, 0, 0
}

func (t *baseTracker) starTracker(ctx context.Context) {

	t.state = &initialRequest{}
	t.state.handle()
}

func (t *baseTracker) stopTracker() {
	t.state.cancel()
}
func (t *baseTracker) resetTracker(duration time.Duration) {
	t.state.reset(duration)
}

type trackerRequestState interface {
	handle()
	cancel()
	reset(duration time.Duration)
}

type initialRequest struct {
	tracker baseTracker
}

func (i *initialRequest) handle() {
	log.Printf("trackerUrl: %v", i.tracker.trackerURL.String())
	log.Printf("sending initial 2 tracker request, trackerType : %v", reflect.TypeOf(i.tracker))
	interval, err := i.tracker.handleRequest()
	if err != nil {
		log.Fatalf("err:  %v",err)
		return
	}
	i.tracker.interval = time.Duration(interval)
	i.tracker.state = &recurringRequest{}
	log.Print("launching tracker request")
	i.tracker.state.handle()

}

func (i *initialRequest) cancel() {

}

func (i *initialRequest) reset(time.Duration) {

}

type recurringRequest struct {
	tracker baseTracker
}

func (r *recurringRequest) handle() {

		r.tracker.timer = time.AfterFunc(r.tracker.interval, func() {
			log.Printf("sending initial tracker request, trackerType : %v", reflect.TypeOf(r.tracker))

			interval, err := r.tracker.handleRequest()
			if err != nil {
				return
			}

			r.tracker.interval = time.Duration(interval)
			r.tracker.state = &recurringRequest{tracker: r.tracker}
			r.handle()
		})

}

func (r *recurringRequest) cancel() {
	r.tracker.timer.Stop()
	r.tracker.state = &initialRequest{}
}

func (r *recurringRequest) reset(duration time.Duration) {
	r.tracker.timer.Stop()
	r.tracker.timer.Reset(duration)
}

type httpTracker2 struct {
	*baseTracker
}

func (trp *httpTracker2) execRequest() (*parser.BMap, error) {
	myID := utils.MyID

	event := trp.getCurrentState()

	//fmt.Printf("%x\n", InfoHash)

	uploaded, downloaded, left := trp.getTransferStats()

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", trp.getInfoHash())
	trackerRequestParam.Add("peer_id", myID)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(downloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	trackerUrl := trp.getAnnouncerUrl()
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

	peers = trp.baseTracker.createPeersFromHTTPTracker(trackerResponse.BLists["peers"].BMaps)

	for _, peer := range peers {
		operation := addPeerOperation{
			peer:  peer,
			swarm: trp.baseTracker.peerManager,
		}
		trp.baseTracker.peerManager.peerOperationReceiver <- operation
	}

	return 0, nil
}

type udpTracker2 struct {
	*baseTracker
}

func (trp *udpTracker2) handleRequest() (int, error) {
	var peers []*Peer
	trackerResponse, err := trp.execRequest()
	if err != nil {
		return 0, nil
	}

	peers = trp.baseTracker.createPeersFromUDPTracker(trackerResponse.PeersAddresses)

	for _, peer := range peers {
		operation := addPeerOperation{
			peer:  peer,
			swarm: trp.baseTracker.peerManager,
		}
		trp.baseTracker.peerManager.peerOperationReceiver <- operation
	}

	return 0, nil
}
func (trp *udpTracker2) execRequest() (*PeerProtocol.UdpMSG, error) {

	/*	// TODO this needs to be moved
		ipByte := make([]byte, 0)
		start := 0
		end := 0
		// just a little fix to know when the we reach the last number of an IP
		ipString := utils.LocalAddr.IP.String() + "."
		for _, c := range ipString {
			if c == '.' {
				p, _ := strconv.Atoi(ipString[start:end])
				ipByte = append(ipByte, byte(p))
				end++
				start = end
			} else {
				end++
			}

		}
		/////////////////////////////////////////

		cleanedUdpAddr := trp.getAnnouncerUrl()
		trackerAddress, _ := net.ResolveUDPAddr("udp", cleanedUdpAddr)

		incomingMsg := make([]byte, 1024)
		var announceResponse UdpMSG
		var announceResponseErr error
		var succeeded bool

		udpConn, udpErr := net.DialUDP("udp", nil, trackerAddress)
		if udpErr == nil {
			randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
			randN := randSeed.Uint32()
			connectionRequest := udpTrackerConnectMsg(UdpMSG{connectionID: udpProtocolID, action: udpConnectRequest, transactionID: int(randN)})

			_, writingErr := udpConn.Write(connectionRequest)

			if writingErr == nil {
				var readingErr error = nil
				connectionTimeCounter := 0
				nByteRead := 0
				connectionSucceed := false
				var connectionResponse UdpMSG
				var connectionResponseErr error = nil

				// waiting for conn response
				for !connectionSucceed && connectionTimeCounter < 4 {
					//fmt.Printf("counter %v\n", connectionTimeCounter)
					_ = udpConn.SetReadDeadline(time.Now().Add(time.Second * 15))
					nByteRead, readingErr = bufio.NewReader(udpConn).Read(incomingMsg)

					if readingErr != nil {
						//println("time out!!!")
						_, writingErr = udpConn.Write(connectionRequest)
						connectionTimeCounter++
					} else {

						connectionSucceed = true
						connectionResponse, connectionResponseErr = parseUdpTrackerResponse(incomingMsg, nByteRead)
					}
					//fmt.Printf("connection response %v nByteRead %v", incomingMsg, nByteRead)

					if connectionSucceed && connectionResponseErr == nil {

						transactionID := randSeed.Uint32()
						key := randSeed.Uint32()
						announceMsg := udpTrackerAnnounceMsg(UdpMSG{connectionID: connectionResponse.connectionID, action: udpAnnounceRequest, transactionID: int(transactionID), infoHash: []byte(peerSwarm.torrent.Downloader.InfoHash), peerID: []byte(utils.MyID), downloaded: peerSwarm.torrent.Downloader.TotalDownloaded, uploaded: peerSwarm.torrent.Downloader.uploaded, left: peerSwarm.torrent.Downloader.left, event: int(atomic.LoadInt32(peerSwarm.torrent.Downloader.State)), ip: ipByte, key: int(key), port: utils.LocalAddr.Port, numWant: -1})

						_, writingErr = udpConn.Write(announceMsg)

						if writingErr == nil {

							hasResponse := false
							announceRequestCountDown := 0
							for !hasResponse && announceRequestCountDown < 4 {
								_ = udpConn.SetReadDeadline(time.Now().Add(time.Second * 15))
								nByteRead, readingErr = bufio.NewReader(udpConn).Read(incomingMsg)

								if readingErr != nil {
									//println("time out!!!")
									_, writingErr = udpConn.Write(announceMsg)
									announceRequestCountDown++
								} else {
									//fmt.Printf("\n response \n %v", incomingMsg)

									hasResponse = true
									announceResponse, announceResponseErr = parseUdpTrackerResponse(incomingMsg, nByteRead)

									if announceResponseErr != nil {
										succeeded = true
									}
								}

							}

						}

						//fmt.Printf("\n Annouce msg byte \n %v \n peer Len %v", announceMsg,len(announceResponse.peersAddresses))

					}

				}

			} else {
				//	fmt.Printf("writing errors")
			}
		}
		//fmt.Printf("\n response \n %v", announceResponse)

		if succeeded {
			return announceResponse, nil
		} else {
			return UdpMSG{}, errors.New("request to baseTracker was unsuccessful. please, try again")
		}
	*/
	return nil, nil
}

func (t *baseTracker) createPeersFromHTTPTracker(peersInfo []*parser.BMap) []*Peer {
	var peers []*Peer
	for _, peerDict := range peersInfo {
		peer := NewPeer(peerDict.Strings["ip"], peerDict.Strings["port"], peerDict.Strings["peer id"])
		peers = append(peers, peer)
		//go peerSwarm.connect(peer)
	}
	return peers
}

func (t *baseTracker) createPeersFromUDPTracker(peersAddress []byte) []*Peer {
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
