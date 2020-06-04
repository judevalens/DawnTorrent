package PeerProtocol

import (
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync/atomic"
	"time"
	"torrent/parser"
	"torrent/utils"
)

const (
	startedEvent   = "started"
	stoppedEvent   = "stopped"
	completedEvent = "completed"
	SubPieceLen    = 16000
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
	PieceCounter int

	done chan bool
}

func NewTorrent(torrentPath string, done chan bool) *Torrent {

	torrent := new(Torrent)
	torrent.File = initTorrentFile(torrent, torrentPath)
	println("torrent.File.infoHash")
	println(torrent.File.infoHash)
	torrent.PeerSwarm = NewPeerSwarm(torrent)
	torrent.requestQueue = NewQueue(torrent)
	torrent.incomingMsgQueue = NewQueue(torrent)
	torrent.requestQueue.Run(5)
	torrent.incomingMsgQueue.Run(5)

	//torrent.worker()
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
	for {

		if torrent.File.currentPiece == nil || torrent.File.SelectNewPiece {
			if torrent.File.behavior == "random" {
				randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
				randN := randomSeed.Intn(torrent.File.NeededPiece.Size())
				randomPiece, found := torrent.File.NeededPiece.Get(start)
				_ = randN
				if found {
					torrent.File.currentPiece = randomPiece.(*Piece)
				}

				torrent.File.SelectNewPiece = false
				torrent.File.CurrentPieceIndex = start
				torrent.PieceCounter = 0
				start++

			} else if torrent.File.behavior == "rarest" {
				//torrent.File.sortPieces()
				//rarestPieceIndex := torrent.File.SortedAvailability[torrent.File.nPiece-1]
				//torrent.File.PiecesMutex.RLock()
				//torrent.File.currentPiece = torrent.File.Pieces[rarestPieceIndex]
				//torrent.File.PiecesMutex.RUnlock()
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

			for i := 0; i < torrent.File.nSubPiece; i++ {

				// 8 subpieces' state is stored in a byte
				subPieceBitMaskIndex := int(math.Ceil(float64(i / 8)))
				subPieceBitIndex := i % 8

				// check the subpiece status by looking at its state in the bitmask
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
							//os.Exit(22929)
						}
					}
					msg := MSG{MsgID: RequestMsg, PieceIndex: torrent.File.currentPiece.PieceIndex, BeginIndex: i * torrent.File.subPieceLen, PieceLen: subPieceLen}

					pieceRequest := GetMsg(msg, nil)




						for j := 0 ; j < len(activeConnection); j++{
							if peersIndex < len(activeConnection){
								randomIndex := randSeed.Intn(len(activeConnection))
							torrent.PeerSwarm.peerMutex.Lock()
							peer := activeConnection[randPeers[randomIndex]]
							torrent.PeerSwarm.peerMutex.Unlock()

							if peer != nil{
								pieceRequest.Peer = peer.(*Peer)
								torrent.requestQueue.Add(pieceRequest)
							}
						}
						}
						peersIndex++

						peersIndex = len(activeConnection)%peersIndex



					//TODO
					// right now we send a piece to all connected peer, we should only be sending requests to peer that has unchocked us
				break
				} else {

					////fmt.Printf("\n%v\n", torrent.File.currentPiece.Pieces[i * torrent.File.SubPieceLen:(i * torrent.File.SubPieceLen)+torrent.File.SubPieceLen])
					//os.Exit(192892)
				}

			}

		}

		//TODO
		// right we select the next piece download randomly
		// after download the first four pieces randomly, we should select the next nth piece based on its rarity
		/*
			if len(torrent.File.completedPieceIndex) > 4 {
				torrent.File.behavior = "rarest"
			}*/

		time.Sleep(time.Second)
	}

}

func (torrent *Torrent) msgRouter(msg MSG) {

	switch msg.MsgID {
	case BitfieldMsg:
		//fmt.Printf("%v", msg.BitfieldRaw)
		// gotta check that bitfield is the correct len

		bitfieldLen := torrent.File.nPiece*8

		if msg.PieceLen == bitfieldLen{
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

						if isPieceAvailable {
							torrent.File.PiecesMutex.Lock()
							torrent.File.Pieces[pieceIndex].Availability++
							if torrent.File.Pieces[pieceIndex].Status != "complete" {
								torrent.File.Pieces[pieceIndex].owners[msg.Peer.id] = msg.Peer

								torrent.PeerSwarm.peerMutex.Lock()
								if !msg.Peer.peerIsInteresting {
									msg.Peer.peerIsInteresting = true
									torrent.requestQueue.Add(GetMsg(MSG{MsgID: InterestedMsg}, msg.Peer))

								}
								torrent.PeerSwarm.peerMutex.Unlock()
							}
							torrent.File.PiecesMutex.Unlock()

						}
						pieceIndex++
						bitIndex--
					}

				}
			}
		}
	case InterestedMsg:
		msg.Peer.interested = true
	case UninterestedMsg:
		msg.Peer.interested = false
	case UnchockeMsg:
		msg.Peer.peerIsChocking = false
	case ChockedMsg:
		msg.Peer.peerIsChocking = true
	case PieceMsg:
		if msg.PieceIndex < torrent.File.nPiece {

			if msg.PieceLen == torrent.File.subPieceLen || msg.PieceLen == torrent.File.Pieces[1].PieceTotalLen%torrent.File.subPieceLen {
				_ = torrent.File.AddSubPiece(msg, msg.Peer)
		}



		}

	case HaveMsg:
		if msg.MsgLen == haveMsgLen && msg.PieceIndex < torrent.File.nPiece{
			torrent.File.PiecesMutex.Lock()
			torrent.File.Pieces[msg.PieceIndex].Availability++
			if torrent.File.Pieces[msg.PieceIndex].Status != "complete" {
				torrent.PeerSwarm.peerMutex.Lock()
				torrent.File.Pieces[msg.PieceIndex].owners[msg.Peer.id] = msg.Peer
				torrent.PeerSwarm.peerMutex.Unlock()
			}
			torrent.File.PiecesMutex.Unlock()
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
