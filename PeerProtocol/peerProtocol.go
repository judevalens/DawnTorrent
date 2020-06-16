package PeerProtocol

import (
	"DawnTorrent/JobQueue"
	"DawnTorrent/parser"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync/atomic"
	"time"
)

const (

	pieceHashLen   = 20

)

type Torrent struct {
	TrackerResponse *parser.Dict
	torrentMetaInfo *parser.Dict
	PeerSwarm       *PeerSwarm
	File            *TorrentFile

	requestQueue     *JobQueue.Queue
	incomingMsgQueue *JobQueue.Queue
	pieceQueue       *JobQueue.Queue
	PieceCounter     int
	chokeCounter int

	done chan bool
}

func NewTorrent(torrentPath string, done chan bool) *Torrent {

	torrent := new(Torrent)
	torrent.File = initTorrentFile(torrent, torrentPath)
	torrent.File.saveTorrent()
	println("torrent.File.infoHash")
	println(torrent.File.infoHash)
	torrent.PeerSwarm = NewPeerSwarm(torrent)
	torrent.requestQueue = JobQueue.NewQueue(torrent)
	torrent.incomingMsgQueue = JobQueue.NewQueue(torrent)
	torrent.pieceQueue = JobQueue.NewQueue(torrent)
	torrent.requestQueue.Run(1)
	torrent.incomingMsgQueue.Run(1)
	torrent.pieceQueue.Run(1)

	//torrent.worker()
//	go torrent.SelectPiece()

time.AfterFunc(time.Second*3, func() {
	go torrent.PieceRequestManager()
})
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
				peer = nil
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

	for {

		if torrent.PeerSwarm.PeerByDownloadRate != nil {
			//torrent.PeerSwarm.SortPeerByDownloadRate()
		}

		if torrent.File.SelectNewPiece {
			if torrent.File.PieceSelectionBehavior == "random" {
						torrent.File.PiecesMutex.Lock()
						torrent.File.CurrentPieceIndex = start
						torrent.File.PiecesMutex.Unlock()
						fmt.Printf("switching piece # %v\n", torrent.File.CurrentPieceIndex)
					start++

				torrent.File.SelectNewPiece = false
				torrent.PieceCounter = 0

			} else if torrent.File.PieceSelectionBehavior == "rarest" {
				//torrent.File.sortPieces()
				//rarestPieceIndex := torrent.File.SortedAvailability[torrent.File.nPiece-1]
				//torrent.File.PiecesMutex.RLock()
				//torrent.File.currentPiece = torrent.File.Pieces[rarestPieceIndex]
				//torrent.File.PiecesMutex.RUnlock()
			}
		}
		time.Sleep(time.Millisecond*25)

	}

}
func ResumeTorrent() *Torrent{
	torrent := new(Torrent)

	return torrent
}

//	Adds a request for piece a request to the a job Queue
//	Params :
//	subPieceRequest *PieceRequest : contains the raw request msg
func (torrent *Torrent)requestPiece(subPieceRequest *PieceRequest)error{
	torrent.PeerSwarm.SortPeerByDownloadRate()
	var err error

	currentPiece := torrent.File.Pieces[torrent.File.CurrentPieceIndex]

	peerIndex := 0

	isPeerFree := false
	var peer *Peer

	for !isPeerFree{
fmt.Printf("looking for a suitable peer | # of available peer : %v\n", torrent.PeerSwarm.PeerByDownloadRate.Size())
	//	fmt.Printf("\nchoking peer # %v\n",torrent.chokeCounter)
	//	fmt.Printf("looking for a suitable peer | # of available peer active connection: %v\n", torrent.PeerSwarm.activeConnection.Size())
		peerI, found := torrent.PeerSwarm.PeerByDownloadRate.Get(peerIndex)

		if found && peerI != nil{
			peer = peerI.(*Peer)
			peerIndex++
		peerIndex = peerIndex%torrent.PeerSwarm.PeerByDownloadRate.Size()

			if peer != nil{
				isPeerFree = peer.isPeerFree()

			//	fmt.Printf("peer is not free, # pendingg Req : %v | peer Index : %v\n ",len(peer.peerPendingRequest),peerIndex)
			}else{
				//fmt.Printf("peer is nil ")

				isPeerFree = false
			}
		}else{
			//fmt.Printf("peer is not found\n")
		}

	}

	if peer != nil{
		subPieceRequest.msg.Peer = peer
	torrent.requestQueue.Add(subPieceRequest.msg)
		peer.peerPendingRequestMutex.Lock()
		peer.peerPendingRequest = append(peer.peerPendingRequest,subPieceRequest)
		peer.peerPendingRequestMutex.Unlock()

	subPieceRequest.timeStamp = time.Now()

	if subPieceRequest.status == nonStartedRequest{
		subPieceRequest.status = pendingRequest
		currentPiece.pendingRequest = append(currentPiece.pendingRequest,subPieceRequest)
	}
		//fmt.Printf("request id %v %v \n",subPieceRequest.startIndex, subPieceRequest.msg)

	}else{
		err = errors.New("request is not sent ")
	}

	return err
}


//	Selects Pieces that need to be downloader
//	When a piece is completely downloaded , a new one is selected
func (torrent *Torrent) PieceRequestManager(){
	start := 0
	torrent.File.timeS = time.Now()

	for  {
		println("223")
		if torrent.File.SelectNewPiece {
			if torrent.File.PieceSelectionBehavior == "random" {
				torrent.File.PiecesMutex.Lock()
				torrent.File.CurrentPieceIndex = start
				torrent.File.PiecesMutex.Unlock()
				fmt.Printf("switching piece # %v\n", torrent.File.CurrentPieceIndex)
				start++

				torrent.File.SelectNewPiece = false
				torrent.PieceCounter = 0

			} else if torrent.File.PieceSelectionBehavior == "rarest" {
				//torrent.File.sortPieces()
				//rarestPieceIndex := torrent.File.SortedAvailability[torrent.File.nPiece-1]
				//torrent.File.PiecesMutex.RLock()
				//torrent.File.currentPiece = torrent.File.Pieces[rarestPieceIndex]
				//torrent.File.PiecesMutex.RUnlock()
			}
		}

		//	store the pieces that is being downloaded
		torrent.File.PiecesMutex.Lock()
		currentPiece := torrent.File.Pieces[torrent.File.CurrentPieceIndex]
		torrent.File.PiecesMutex.Unlock()


		currentPiece.pendingRequestMutex.Lock()
		nSubPieceRequest := len(currentPiece.pendingRequest)
		currentPiece.pendingRequestMutex.Unlock()


		//	the number of subPiece request shall not exceed maxRequest
		//	new requests are sent if the previous request have been fulfilled
		if nSubPieceRequest < maxRequest{
			nReq := maxRequest-len(currentPiece.pendingRequest)


			nReq = int(math.Min(float64(nReq), float64(len(currentPiece.neededSubPiece))))

			i := 0
			randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

			for i < nReq{
				// 	subPieces request are randomly selected
				randomN := randomSeed.Intn(len(currentPiece.neededSubPiece))

				currentPiece.pendingRequestMutex.Lock()
				req := currentPiece.neededSubPiece[randomN]

				//fmt.Printf("sending request\n")
				err := torrent.requestPiece(req)

				// once a request is added to the job JobQueue, it is removed from the needed subPiece / prevents from being reselected
				if err == nil{
					currentPiece.neededSubPiece[randomN] = currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1]
					currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1] = nil
					currentPiece.neededSubPiece = currentPiece.neededSubPiece[:len(currentPiece.neededSubPiece)-1]
				}
				currentPiece.pendingRequestMutex.Unlock()
				i++
			}

		}
		currentPiece.pendingRequestMutex.Lock()

		// re-sends request that have been pending for a certain amounts of time
		nPendingRequest := len(currentPiece.pendingRequest)

		for r := 0; r < nPendingRequest; r++{
			pendingRequest := currentPiece.pendingRequest[r]


			if time.Now().Sub(pendingRequest.timeStamp) >= maxWaitingTime{
				//fmt.Printf("sending request\n")
		go torrent.requestPiece(pendingRequest)
			}
		}
		currentPiece.pendingRequestMutex.Unlock()

		time.Sleep(time.Millisecond*25)
	}
}

//	Receives msg from other peers and calls the appropriate methods
func (torrent *Torrent) msgRouter(msg *MSG) {

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
		} else {
			//fmt.Printf("correctlen %v actual Len %v", bitfieldCorrectLen, len(msg.BitfieldRaw))
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
		if msg.MsgLen == unChokeMsgLen{
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
		if msg.PieceIndex < torrent.File.nPiece {
			// verifies that the length of the data is not greater or smaller than amount requested
			if msg.PieceLen == torrent.File.subPieceLen || msg.PieceLen == torrent.File.Pieces[msg.PieceIndex].Len%torrent.File.subPieceLen {
				_ = torrent.File.AddSubPiece(msg, msg.Peer)
			}
		}

	case HaveMsg:
		if msg.MsgLen == haveMsgLen && msg.PieceIndex < torrent.File.nPiece {
			torrent.File.pieceAvailabilityMutex.Lock()
			torrent.File.Pieces[msg.PieceIndex].Availability++
			torrent.File.pieceAvailabilityMutex.Unlock()
			torrent.File.SortPieceByAvailability()

		}

	}

}

func (torrent *Torrent) Worker(queue *JobQueue.Queue, id int) {
	for {
		request := queue.Pop()
		switch item := request.(type) {

		case *MSG:

			switch item.msgType {
			case outgoingMsg:
				torrent.PeerSwarm.peerMutex.Lock()
				if item.Peer != nil {
					_, _ = item.Peer.connection.Write(item.rawMsg)
				}
				torrent.PeerSwarm.peerMutex.Unlock()


			case incomingMsg:
				torrent.msgRouter(item)
			}

		}
	}
}
