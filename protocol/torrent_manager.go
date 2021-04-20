package protocol

import (
	"DawnTorrent/PeerProtocol"
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	started   = iota
	stopped   = iota
	completed = iota
)

const (
	httpTracker = iota
	udpTracker  = iota
)

type TorrentManager struct {
	torrent            *Torrent
	peerSwam           *PeerSwarm
	trackerRequestChan *time.Ticker
	stopTrackerRequest chan interface{}
	torrentState       int
	uploaded           int
	totalDownloaded    int
	left               int
	stateChan          chan int
	state              int
	trackerType        int
}

func (manager TorrentManager) init() {

	for {
		manager.state = <-manager.stateChan

		switch manager.state {
		case started:
			go manager.runPeriodicTracker(manager.trackerType)
			manager.trackerRequestChan = time.NewTicker(time.Nanosecond)
		case stopped:
			manager.trackerRequestChan.Stop()
		case completed:
			//TODO do something !
		}
	}

}

func (manager *TorrentManager) runTracker(trackerType int) {
	var peers []*Peer
	if trackerType == httpTracker {
		trackerResponse, err := manager.sendHTTPTrackerRequest(1, 0, 0, 0, "", "")

		if err != nil {
			log.Fatal(err)
		}

		peers = manager.createPeersFromHTTPTracker(trackerResponse.BLists["peers"].BMaps)
	} else {

		trackerResponse, err := manager.sendUDPTrackerRequest(1)
		if err != nil{
			log.Fatal(err)
		}
		peers = manager.createPeersFromUDPTracker(trackerResponse.PeersAddresses)

	}

	// add those peers to swarm
	for _, peer := range peers {
		operation := addPeerOperation{
			peer:  peer,
			swarm: manager.peerSwam,
		}
		manager.peerSwam.peerOperation <- operation
	}
}

func (manager *TorrentManager) runPeriodicTracker(trackerType int, ) {
	println("sending to tracker ................")

	for {
		<-manager.trackerRequestChan.C
		manager.runTracker(trackerType)

	}

}

func (manager *TorrentManager) runPeriodicDownloader(){

}

func (torrent *Torrent) msgRouter(msg *PeerProtocol.MSG) {

	switch msg.MsgID {
	case BitfieldMsg:
		// gotta check that bitfield is the correct len
		bitfieldCorrectLen := int(math.Ceil(float64(torrent.Downloader.nPiece) / 8.0))

		counter := 0
		if (len(msg.BitfieldRaw)) == bitfieldCorrectLen {

			pieceIndex := 0

			for i := 0; i < torrent.Downloader.nPiece; i += 8 {
				bitIndex := 7
				currentByte := msg.BitfieldRaw[i/8]
				for bitIndex >= 0 && pieceIndex < torrent.Downloader.nPiece {
					counter++
					currentBit := uint8(math.Exp2(float64(bitIndex)))
					bit := currentByte & currentBit

					isPieceAvailable := bit != 0
					torrent.PeerSwarm.peerMutex.Lock()

					//TODO if a peer is removed, it is a problem if we try to access it
					// need to add verification that the peer is still in the map
					peer, isPresent := torrent.PeerSwarm.PeersMap.Get(msg.Peer.id)
					if isPresent {
						peer.(*Peer).AvailablePieces[pieceIndex] = isPieceAvailable
					}
					torrent.PeerSwarm.peerMutex.Unlock()

					// if piece available we put in the sorted map

					if isPieceAvailable {

						torrent.Downloader.pieceAvailabilityMutex.Lock()
						torrent.Downloader.Pieces[pieceIndex].Availability++
						torrent.Downloader.pieceAvailabilityMutex.Unlock()
					}
					pieceIndex++
					bitIndex--
				}

			}
		} else {
			//fmt.Printf("correctlen %v actual Len %v", bitfieldCorrectLen, len(msg.BitfieldRaw))
		}

		torrent.Downloader.SortPieceByAvailability()

	case InterestedMsg:
		torrent.PeerSwarm.peerMutex.Lock()
		msg.Peer.interested = true
		torrent.PeerSwarm.peerMutex.Unlock()

	case UninterestedMsg:
		torrent.PeerSwarm.peerMutex.Lock()
		msg.Peer.interested = false
		torrent.PeerSwarm.peerMutex.Unlock()

	case UnchockeMsg:
		if msg.MsgLen == unChokeMsgLen {
			torrent.PeerSwarm.peerMutex.Lock()
			msg.Peer.peerIsChocking = false
			torrent.PeerSwarm.peerMutex.Unlock()
		}

	case ChockedMsg:
		if msg.MsgLen == chokeMsgLen {
			torrent.PeerSwarm.peerMutex.Lock()
			msg.Peer.peerIsChocking = true
			torrent.chokeCounter++

			torrent.PeerSwarm.peerMutex.Unlock()

		}
	case PieceMsg:
		// making sure that we are receiving a valid piece index
		if msg.PieceIndex < torrent.Downloader.nPiece {
			// verifies that the length of the data is not greater or smaller than amount requested
			if msg.PieceLen == torrent.Downloader.subPieceLength || msg.PieceLen == torrent.Downloader.Pieces[msg.PieceIndex].Len%torrent.Downloader.subPieceLength {
				//	_ = torrent.Downloader.AddSubPiece(msg, msg.Peer)
				torrent.Downloader.addPieceChannel <- msg
			}
		}

	case HaveMsg:
		if msg.MsgLen == haveMsgLen && msg.PieceIndex < torrent.Downloader.nPiece {
			torrent.Downloader.pieceAvailabilityMutex.Lock()
			torrent.Downloader.Pieces[msg.PieceIndex].Availability++
			torrent.Downloader.pieceAvailabilityMutex.Unlock()
			torrent.Downloader.SortPieceByAvailability()

		}

	}

}


func (manager *TorrentManager) sendHTTPTrackerRequest(state int, uploaded, totalDownloaded, left int, infoHash string, announcerURL string) (*parser.BMap, error) {

	myID := utils.MyID

	event := ""
	switch state {
	case 0:
		event = "started"
	case 1:
		event = "stopped"
	case 2:
		event = "completed"

	}

	//fmt.Printf("%x\n", InfoHash)

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", infoHash)
	trackerRequestParam.Add("peer_id", myID)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(totalDownloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	trackerUrl := announcerURL + "?" + trackerRequestParam.Encode()
	fmt.Printf("\n Param \n %v \n", trackerUrl)

	trackerResponseByte, _ := http.Get(trackerUrl)
	trackerResponse, _ := ioutil.ReadAll(trackerResponseByte.Body)

	fmt.Printf("%v\n", string(trackerResponse))

	trackerDictResponse := parser.UnmarshallFromArray(trackerResponse)

	requestErr, isPresent := trackerDictResponse.Strings["failure reason"]

	if isPresent {
		log.Fatalf("error: %v", errors.New(requestErr))
	}

	return nil, nil

}

func (manager *TorrentManager) sendUDPTrackerRequest(event int) (*PeerProtocol.UdpMSG, error) {

	/*
		// TODO this needs to be moved
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

		cleanedUdpAddr := peerSwarm.torrent.Downloader.Announce[6:]
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
			return UdpMSG{}, errors.New("request to tracker was unsuccessful. please, try again")
		}

	*/
	return nil, nil
}

func (manager *TorrentManager) createPeersFromHTTPTracker(peersInfo []*parser.BMap) []*Peer {
	var peers []*Peer
	for _, peerDict := range peersInfo {
		peer := NewPeer(peerDict.Strings["ip"], peerDict.Strings["port"], peerDict.Strings["peer id"])
		peers = append(peers, peer)
		//go peerSwarm.connect(peer)
	}
	return peers
}

func (manager *TorrentManager) createPeersFromUDPTracker(peersAddress []byte) []*Peer {
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



