package PeerProtocol

import (
	"bytes"
	"fmt"
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
	File            *torrentFile

	requestQueue     *Queue
	incomingMsgQueue *Queue

	done chan bool
}

func NewTorrent(torrentPath string, done chan bool) *Torrent {

	torrent := new(Torrent)
	torrent.File = NewFile(torrent, torrentPath)
	println("torrent.File.infoHash")
	println(torrent.File.infoHash)
	torrent.PeerSwarm = NewPeerSwarm(torrent)
	torrent.requestQueue = NewQueue(torrent)
	torrent.incomingMsgQueue = NewQueue(torrent)
	torrent.requestQueue.Run(4)
	torrent.incomingMsgQueue.Run(4)

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
	for {

		if torrent.File.currentPiece == nil || torrent.File.currentPiece.Status == "complete" {
			if torrent.File.behavior == "random" {

				randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
				randN := randomSeed.Intn(len(torrent.File.neededPieceMap))
				i := 0
				for k := range torrent.File.neededPieceMap {
					torrent.File.currentPiece = torrent.File.neededPieceMap[k]

					if i > randN {
						break
					}
				}
			} else if torrent.File.behavior == "rarest" {
				torrent.File.sortPieces()
				rarestPieceIndex := torrent.File.SortedAvailability[torrent.File.nPiece-1]
				torrent.File.PiecesMutex.RLock()
				torrent.File.currentPiece = torrent.File.Pieces[rarestPieceIndex]
				torrent.File.PiecesMutex.RUnlock()

			}
			torrent.File.currentPiece.Status = "inProgress"
		}
		if torrent.File.currentPiece.Status == "inProgress" {
			for i, _ := range torrent.File.currentPiece.SubPieces {
				if len(torrent.File.currentPiece.SubPieces[i]) == 0 {

					torrent.PeerSwarm.activeConnectionMutex.Lock()

					for _, o := range torrent.PeerSwarm.activeConnection.Values() {

						msg := MSG{MsgID: RequestMsg, PieceIndex: uint32(torrent.File.currentPiece.PieceIndex), BeginIndex: uint32(i * torrent.File.SubPieceLen), PieceLen: uint32(torrent.File.SubPieceLen)}
						p, found := torrent.PeerSwarm.activeConnection.Get(o.(string))

						if found {
							_, _ = p.(*Peer).connection.Write(GetMsg(MSG{MsgID: InterestedMsg}, p.(*Peer)).rawMsg)

							_, _ = p.(*Peer).connection.Write(GetMsg(msg, p.(*Peer)).rawMsg)
						}
						fmt.Printf("owner %v \nsub_index %v index %v beginIndex %v len %v\n", p.(*Peer).id, i, torrent.File.currentPiece.PieceIndex, i*torrent.File.SubPieceLen, torrent.File.SubPieceLen)

					}
					torrent.PeerSwarm.activeConnectionMutex.RUnlock()

					//break
				}

			}

		}
		/// change the states here

		if len(torrent.File.completedPieceIndex) > 4 {
			torrent.File.behavior = "rarest"
		}
		time.Sleep(time.Second)
	}

}

func (torrent *Torrent) msgRouter(msg MSG) {

	switch msg.MsgID {
	case BitfieldMsg:
		fmt.Printf("%v", msg.BitfieldRaw)
		// gotta check that bitfield is the correct len
		bitfieldCorrectLen := int(math.Ceil(float64(torrent.File.nPiece) / 8.0))
		counter := 0
		if (len(msg.BitfieldRaw)) == bitfieldCorrectLen {

			pieceIndex := 0

			for i := 0; i < len(torrent.File.Pieces); i += 8 {
				bitIndex := 7
				currentByte := msg.BitfieldRaw[i/8]
				for bitIndex >= 0 && pieceIndex < len(torrent.File.Pieces) {
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
					fmt.Printf("piece index %v is %v for byte %v \n", pieceIndex, isPieceAvailable, currentByte)

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
	case InterestedMsg:
		msg.Peer.updateState(msg.Peer.chocked, true, torrent)
	case UnchockeMsg:
		msg.Peer.peerIsChocking = false
	case PieceMsg:
		_ = torrent.addSubPiece(msg, msg.Peer)
		////////////os.Exit(7887)
	case ChockedMsg:
		msg.Peer.peerIsChocking = true

	case HaveMsg:
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

func (torrent *Torrent) saveTorrent() {
}

// assembles a complete piece
// writes piece to the file once completed
func (torrent *Torrent) addSubPiece(msg MSG, peer *Peer) error {
	var err error = nil

	subPieceIndex := int(math.Ceil(float64(msg.BeginIndex / SubPieceLen)))

	torrent.File.Pieces[msg.PieceIndex].CurrentLen += int(msg.PieceLen)

	torrent.File.Pieces[msg.PieceIndex].SubPieces[subPieceIndex] = make([]byte, int(msg.PieceLen))
	copy(torrent.File.Pieces[msg.PieceIndex].SubPieces[subPieceIndex], msg.Piece)

	// piece is complete add it to the file
	if torrent.File.Pieces[msg.PieceIndex].CurrentLen == torrent.File.Pieces[msg.PieceIndex].PieceTotalLen {
		torrent.File.Pieces[msg.PieceIndex].Status = "complete"
		file, _ := os.OpenFile(utils.DawnTorrentHomeDir+"/"+torrent.File.torrentMetaInfo.MapDict["info"].MapString["name"], os.O_CREATE|os.O_RDWR, os.ModePerm)

		delete(torrent.File.neededPieceMap, int(msg.PieceIndex))

		completePiece := bytes.Join(torrent.File.Pieces[msg.PieceIndex].SubPieces, []byte{})
		_, _ = file.WriteAt(completePiece, int64(msg.PieceIndex)*int64(torrent.File.PieceLen))

		// when a piece is complete, we update the index list
	} else {
		torrent.File.Pieces[msg.PieceIndex].Status = "inProgress"
	}
	peer.numByteDownloaded += int(msg.PieceLen)
	peer.lastTimeStamp = time.Now()
	peer.time += time.Now().Sub(peer.lastTimeStamp).Seconds()
	peer.DownloadRate = (float64(peer.numByteDownloaded)) / peer.time
	return err
}

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
					fmt.Printf("send to %v by woker # %v\n", item.Peer.connection.RemoteAddr().String(), id)
				} else {
					fmt.Printf("errr %v", err)
					//os.Exit(22)
				}
			case incomingMsg:
				torrent.msgRouter(item)
			}

		}
	}
}
