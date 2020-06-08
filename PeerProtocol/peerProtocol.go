package PeerProtocol

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync/atomic"
	"time"
	"DawnTorrent/parser"
	"DawnTorrent/utils"
)

const (
	startedEvent   = "started"
	stoppedEvent   = "stopped"
	completedEvent = "completed"
	SubPieceLen    = 16384
	maxWaitingTime = time.Millisecond*250
	pieceHashLen   = 20
	priority1      = 0
	priority2      = -1
	priority3      = -2
	incomingMsg    = 1
	outgoingMsg    = -1
)

type Torrent struct {
	TrackerResponse *parser.Dict
	torrentMetaInfo *parser.Dict
	PeerSwarm       *PeerSwarm
	File            *TorrentFile

	requestQueue     *Queue
	incomingMsgQueue *Queue
	pieceQueue *Queue
	PieceCounter int

	done chan bool
}

func NewTorrent(torrentPath string, done chan bool) *Torrent {

	torrent := new(Torrent)
	torrent.File = initTorrentFile(torrent, torrentPath)
	println("DawnTorrent.File.infoHash")
	println(torrent.File.infoHash)
	torrent.PeerSwarm = NewPeerSwarm(torrent)
	torrent.requestQueue = NewQueue(torrent)
	torrent.incomingMsgQueue = NewQueue(torrent)
	torrent.pieceQueue = NewQueue(torrent)
	torrent.requestQueue.Run(1)
	torrent.incomingMsgQueue.Run(1)
	torrent.pieceQueue.Run(1)

	//DawnTorrent.worker()
	go torrent.SelectPiece()

	return torrent
}

func (torrent *Torrent) TitForTat() {

	OUCounter := 0

	for true {
		torrent.PeerSwarm.peerMutex.Lock()
		sort.Sort(torrent.PeerSwarm)
		torrent.PeerSwarm.peerMutex.Unlock()

		numUnchockedPeer := 0
		i := 0
		var peer *Peer
		for numUnchockedPeer < 3 {
			isPeerInterested := false
			for !isPeerInterested && i < int(atomic.LoadInt32(torrent.PeerSwarm.nActiveConnection)) {
				peer = torrent.PeerSwarm.GetPeerByDownloadRate(numUnchockedPeer)
				isPeerInterested = peer.interested
				i++
			}
			torrent.PeerSwarm.peerMutex.RLock()

			if peer != nil {
				torrent.PeerSwarm.unChockedPeer[numUnchockedPeer].chocked = true
				torrent.PeerSwarm.unChockedPeer[numUnchockedPeer] = peer
				torrent.PeerSwarm.unChockedPeer[numUnchockedPeer].chocked = false
			}
			torrent.PeerSwarm.peerMutex.RUnlock()

			i++
			numUnchockedPeer++
		}

		if OUCounter%3 == 0 {
			randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
			//before selecting a peer for optimistic unchoking
			// we need to have a least one chocked interested peer

			interestedPeerMapLen := len(torrent.PeerSwarm.interestedPeerMap)

			if interestedPeerMapLen > 0 {
				randN := randomSeed.Intn(len(torrent.PeerSwarm.interestedPeerMap))
				r := 0
				var randomPeer *Peer
				for k := range torrent.PeerSwarm.interestedPeerMap {
					randomPeer = torrent.PeerSwarm.interestedPeerMap[k]
					if r > randN {
						break
					}
					r++
				}
				if torrent.PeerSwarm.unChockedPeer[3] != nil {
					torrent.PeerSwarm.unChockedPeer[3].updateState(true, true, torrent)
				}
				torrent.PeerSwarm.unChockedPeer[3] = randomPeer
				torrent.PeerSwarm.unChockedPeer[3].chocked = false
			}

		}

		OUCounter++

		d, _ := time.ParseDuration("10s")
		time.Sleep(d)
	}

}
func (torrent *Torrent) SelectPiece() {
	start := 0
	torrent.File.timeS = time.Now()

	for{
		if torrent.File.currentPiece == nil || torrent.File.SelectNewPiece {
			if torrent.File.behavior == "random" {
				randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
				randN := randomSeed.Intn(torrent.File.NeededPiece.Size())


				foundPiece := false


				for !foundPiece{
					torrent.File.SortPieceByAvailability()




					torrent.File.pieceAvailabilityMutex.Lock()
					mostAvailablePieceIndexI, _ := torrent.File.PieceAvailability.Get(start)
					torrent.File.pieceAvailabilityMutex.Unlock()

					mostAvailablePieceIndex := mostAvailablePieceIndexI.(*Piece).PieceIndex
					pieceByAvailabilityI, found := torrent.File.NeededPiece.Get(mostAvailablePieceIndex)
					_ = randN
					if found {
						pieceByAvailability :=	pieceByAvailabilityI.(*Piece)
						torrent.File.currentPiece = pieceByAvailability
						foundPiece = true
						fmt.Printf(" mostAvailablePieceIndexI %v \n", mostAvailablePieceIndex)
						torrent.File.CurrentPieceIndex = mostAvailablePieceIndex
					}
					start++

				}

				torrent.File.SelectNewPiece = false
				torrent.PieceCounter = 0

			} else if torrent.File.behavior == "rarest" {
				//DawnTorrent.File.sortPieces()
				//rarestPieceIndex := DawnTorrent.File.SortedAvailability[DawnTorrent.File.nPiece-1]
				//DawnTorrent.File.PiecesMutex.RLock()
				//DawnTorrent.File.currentPiece = DawnTorrent.File.Pieces[rarestPieceIndex]
				//DawnTorrent.File.PiecesMutex.RUnlock()
			}
			torrent.File.currentPiece.Status = "inProgress"
		}
		if !torrent.File.SelectNewPiece {
			// the index of the current file is stored in the piece that is being download

			// iterate through the sub pieces of piece
			// nSubPiece = piece/subPieceLe. subPieceLen is arbitrary

			torrent.PeerSwarm.activeConnectionMutex.Lock()
			activeConnection := torrent.PeerSwarm.activeConnection.Values()
			torrent.PeerSwarm.activeConnectionMutex.Unlock()

			randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

			randPeers := randSeed.Perm(len(activeConnection))

			peersIndex := 0
			requestSent := 0
			currentPiece := torrent.File.Pieces[torrent.File.CurrentPieceIndex]
			for i := 0; i < torrent.File.nSubPiece; i++ {

				// 8 subpieces' state is stored in a byte
				subPieceBitMaskIndex := int(math.Ceil(float64(i / 8)))
				subPieceBitIndex := i % 8

				// check the sub piece status by looking at its state in the bitmask
				// if bit is off, we send a request to the connected peer

				torrent.File.PiecesMutex.Lock()
				isBitOff := !utils.IsBitOn(torrent.File.currentPiece.subPieceMask[subPieceBitMaskIndex], subPieceBitIndex)
				torrent.File.PiecesMutex.Unlock()
				if isBitOff {
					subPieceLen := 0
					subPieceLen = torrent.File.subPieceLen
					if i == torrent.File.nSubPiece-1 {
						if torrent.File.pieceLength%torrent.File.subPieceLen != 0 {
							subPieceLen = torrent.File.pieceLength % torrent.File.subPieceLen
						}
					}
					beginIndex := i * torrent.File.subPieceLen
					msg := MSG{MsgID: RequestMsg, PieceIndex: torrent.File.CurrentPieceIndex, BeginIndex: beginIndex, PieceLen: subPieceLen}

					pieceRequest := GetMsg(msg, nil)
					currentPiece.pendingRequestMutex.Lock()
					blockRequestInterface, foundBlockRequest := currentPiece.pendingRequest.Get(beginIndex)
					currentPiece.pendingRequestMutex.Unlock()

					if len(activeConnection) > 0 {

						if foundBlockRequest {

							blockRequest := blockRequestInterface.(*pendingRequest)
							if time.Now().Sub(blockRequest.timeStamp) > maxWaitingTime {
								///	fmt.Printf("found request for %v\n", beginIndex)
								var peer *Peer
								counter := 0
								randN := randSeed.Intn(len(activeConnection))

								peer = activeConnection[randN].(*Peer)
								counter++

								if peer != nil {

									pieceRequest.Peer = peer
									torrent.pieceQueue.Add(pieceRequest)

									blockRequest.timeStamp = time.Now()
									requestSent++
								}

							}
						} else {
							//	fmt.Printf("active Peer %v\n", DawnTorrent.PeerSwarm.activeConnection.Size())
							pieceRequest.Peer = activeConnection[randPeers[peersIndex]].(*Peer)

							c := 0
							for c < 5 && c < len(activeConnection) {
								pieceRequest.Peer = activeConnection[randPeers[peersIndex]].(*Peer)
								torrent.pieceQueue.Add(pieceRequest)
								peersIndex++
								peersIndex = len(activeConnection) % peersIndex
								c++
								requestSent++
							}

							//DawnTorrent.pieceQueue.Add(pieceRequest)
							pendingRequest := new(pendingRequest)
							pendingRequest.peerID = pieceRequest.Peer.id
							pendingRequest.startIndex = beginIndex
							pendingRequest.timeStamp = time.Now()

							currentPiece.pendingRequestMutex.Lock()
							currentPiece.pendingRequest.Put(beginIndex, pendingRequest)
							currentPiece.pendingRequestMutex.Unlock()
							//fmt.Printf("adding request for %v\n", beginIndex)
							peersIndex++
							peersIndex = len(activeConnection) % peersIndex

						}

					} else {
						println("not enough peer")
					}

				}
			}

				}


		}

		//TODO
		// right we select the next piece download randomly
		// after download the first four pieces randomly, we should select the next nth piece based on its rarity
		/*
			if len(DawnTorrent.File.completedPieceIndex) > 4 {
				DawnTorrent.File.behavior = "rarest"
			}*/


}

func (torrent *Torrent) msgRouter(msg MSG) {

	switch msg.MsgID {
	case BitfieldMsg:
		// gotta check that bitfield is the correct len
		bitfieldCorrectLen := int(math.Ceil(float64(torrent.File.nPiece) / 8.0))

			counter := 0
			if (len(msg.BitfieldRaw)) == bitfieldCorrectLen {

				pieceIndex := 0

				for i := 0; i < torrent.File.nPiece; i += 8 {
					bitIndex := 7
					currentByte := msg.BitfieldRaw[i/8]
					for bitIndex >= 0 && pieceIndex < torrent.File.nPiece {
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

							torrent.File.pieceAvailabilityMutex.Lock()
							torrent.File.Pieces[pieceIndex].Availability++
							torrent.File.pieceAvailabilityMutex.Unlock()
						}
						pieceIndex++
						bitIndex--
					}

				}
			}else{
				fmt.Printf("correctlen %v actual Len %v",bitfieldCorrectLen,len(msg.BitfieldRaw) )
		}


		torrent.File.SortPieceByAvailability()


	case InterestedMsg:
		torrent.PeerSwarm.peerMutex.Lock()
		msg.Peer.interested = true
		torrent.PeerSwarm.peerMutex.Unlock()

	case UninterestedMsg:
		torrent.PeerSwarm.peerMutex.Lock()
		msg.Peer.interested = false
		torrent.PeerSwarm.peerMutex.Unlock()

	case UnchockeMsg:
		torrent.PeerSwarm.peerMutex.Lock()
		msg.Peer.peerIsChocking = false
		torrent.PeerSwarm.peerMutex.Unlock()

	case ChockedMsg:
		torrent.PeerSwarm.peerMutex.Lock()
		msg.Peer.peerIsChocking = true
		torrent.PeerSwarm.peerMutex.Unlock()
	case PieceMsg:
		// making sure that we are receiving a valid piece index
		if msg.PieceIndex < torrent.File.nPiece {
			// verifies that the length of the data is not greater or smaller than amount requested
			if msg.PieceLen == torrent.File.subPieceLen || msg.PieceLen == torrent.File.Pieces[msg.PieceLen].Len%torrent.File.subPieceLen {
				_ = torrent.File.AddSubPiece(msg, msg.Peer)
		}
		}

	case HaveMsg:
		if msg.MsgLen == haveMsgLen && msg.PieceIndex < torrent.File.nPiece{
			torrent.File.pieceAvailabilityMutex.Lock()
			torrent.File.Pieces[msg.PieceIndex].Availability++
			torrent.File.pieceAvailabilityMutex.Unlock()
			torrent.File.SortPieceByAvailability()

		}

	}

}

func (torrent *Torrent) saveTorrent() {
}

// assembles a complete piece
// writes piece to the file once completed

func (torrent *Torrent) savePieces() {

	_, fileErr := os.OpenFile(utils.DawnTorrentHomeDir+"/"+torrent.torrentMetaInfo.MapDict["info"].MapString["name"], os.O_CREATE|os.O_RDWR, os.ModePerm)

	if fileErr != nil {
		log.Fatal(fileErr)
	}
}

func (torrent *Torrent) Worker(queue *Queue, id int) {
	for {
		request := queue.Pop()
		switch item := request.(type) {

		case MSG:

			switch item.msgType {
			case outgoingMsg:
				_, err := item.Peer.connection.Write(item.rawMsg)
				if err == nil {
					//fmt.Printf("send to %v by woker # %v\n", item.Peer.connection.RemoteAddr().String(), id)
				} else {
					//fmt.Printf("errr %v", err)
					//os.Exit(22)
				}
			case incomingMsg:
				torrent.msgRouter(item)
			}

		}
	}
}
