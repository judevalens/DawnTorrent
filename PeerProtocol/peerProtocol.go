package PeerProtocol

import (
	"bytes"
	"errors"
	"fmt"
	binaryHeap "github.com/emirpasic/gods/trees/binaryheap"
	"log"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"
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
)

type Torrent struct {
	TrackerResponse   *parser.Dict
	torrentMetaInfo   *parser.Dict
	PeerSwarm         *PeerSwarm
	File              *torrentFile
	RequestQueue      *binaryHeap.Heap
	incomingMsgQueue  *binaryHeap.Heap
	blockRequestQueue sync.WaitGroup
	requestQueueMutex sync.Mutex
	requestQueueAtomic  *int32
	RequestQueueBlockMutex sync.Mutex
	incomingMsgQueueAtomic *int32
	incomingMsgQueueMutex *int32


	done chan bool
}

func NewTorrent(torrentPath string,done chan bool) *Torrent {
	torrent := new(Torrent)
	torrent.RequestQueue = binaryHeap.NewWith(requestQueueComparator)
	torrent.requestQueueAtomic= new(int32)
	atomic.AddInt32(torrent.requestQueueAtomic,0)
	torrent.incomingMsgQueue = binaryHeap.NewWith(requestQueueComparator)

	torrent.incomingMsgQueueAtomic= new(int32)
	atomic.AddInt32(torrent.incomingMsgQueueAtomic,0)

	torrent.File = NewFile(torrent,torrentPath)
	println("torrent.File.infoHash")
	println(torrent.File.infoHash)
	torrent.PeerSwarm = NewPeerSwarm(torrent)

	torrent.worker()
	torrent.SelectPiece()
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
			isPresent := false
			for !isPresent && i < len(torrent.PeerSwarm.Peers) {
				peerIndex := torrent.PeerSwarm.SortedPeersIndex[(len(torrent.PeerSwarm.SortedPeersIndex) - 1 - i)]
				peer = torrent.PeerSwarm.Peers[peerIndex]
				_, isPresent = torrent.PeerSwarm.interestedPeerMap[peer.id]
				i++
			}
			torrent.PeerSwarm.peerMutex.Lock()

			if peer != nil {
				delete(torrent.PeerSwarm.interestedPeerMap, peer.id)
				if torrent.PeerSwarm.unChockedPeer[numUnchockedPeer] != nil {
					torrent.PeerSwarm.unChockedPeer[numUnchockedPeer].updateState(true, true, torrent)
				}
				torrent.PeerSwarm.unChockedPeer[numUnchockedPeer] = peer
				torrent.PeerSwarm.unChockedPeer[numUnchockedPeer].chocked = false
			}
			torrent.PeerSwarm.peerMutex.Unlock()

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

		if  torrent.File.currentPiece == nil || torrent.File.currentPiece.Status == "complete"{
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
			}else if torrent.File.behavior == "rarest" {
				torrent.File.sortPieces()
				rarestPieceIndex   := torrent.File.SortedAvailability[torrent.File.nPiece-1]
				torrent.File.currentPiece = torrent.File.Pieces[rarestPieceIndex]
			}
			torrent.File.currentPiece.Status = "inProgress"

		}
		if torrent.File.currentPiece.Status == "inProgress" {
			for i,_ := range torrent.File.currentPiece.SubPieces{
				if len(torrent.File.currentPiece.SubPieces[i]) == 0 {
					for _, o := range torrent.File.currentPiece.owners{
						if !o.peerChocking{
							println(torrent.File.currentPiece.SubPieceLen)
							println(i*torrent.File.currentPiece.SubPieceLen)
							println(torrent.File.fileLen)
							msg := MSG{MsgID: RequestMsg,PieceIndex: uint32(torrent.File.currentPiece.PieceIndex),BeginIndex: uint32(i*torrent.File.SubPieceLen),PieceLen: uint32(torrent.File.SubPieceLen)}
							req :=Request{msg: GetMsg(msg),Peer: o,Priority: 2}
							torrent.AddRequest(req)

							fmt.Printf("owner %v \nsub_index %v index %v beginIndex %v len %v\n",o.id,i, torrent.File.currentPiece.PieceIndex,i*torrent.File.SubPieceLen,torrent.File.SubPieceLen)
						}
					}
			break
				}

			}

		}
		/// change the states here

		if len(torrent.File.completedPieceIndex) > 4{
			torrent.File.behavior = "rarest"
		}
		time.Sleep(time.Second)
	}

}

func (torrent *Torrent) msgRouter(msg MSG) {

	switch msg.MsgID {
	case BitfieldMsg:
		// gotta check that bitfield is the correct len
		bitfieldCorrectLen := int(math.Ceil(float64(torrent.File.nPiece) / 8.0))
		counter := 0
		if (len(msg.BitfieldRaw)) == bitfieldCorrectLen {

			pieceIndex := 0

			for i := 0; i < len(torrent.File.Pieces) ; i+=8 {
				bitIndex := 7
				currentByte:= msg.BitfieldRaw[i/8]
				for bitIndex >= 0 && pieceIndex < len(torrent.File.Pieces){
					counter++
					currentBit := uint8(math.Exp2(float64(bitIndex)))
					bit := currentByte & currentBit

					isPieceAvailable := bit != 0
					torrent.File.PiecesMutex.Lock()
					torrent.PeerSwarm.peerMutex.Lock()

					//TODO if a peer is removed, it is a problem if we try to access it
					// need to add verification that the peer is still in the map
					_, isPresent := torrent.PeerSwarm.PeersMap[msg.Sender.id]

					if isPresent {
						torrent.PeerSwarm.PeersMap[msg.Sender.id].AvailablePieces[pieceIndex] = isPieceAvailable
					}
					torrent.PeerSwarm.peerMutex.Unlock()
					if isPieceAvailable {
						torrent.File.Pieces[pieceIndex].Availability++
						if torrent.File.Pieces[pieceIndex].Status != "complete" {
							torrent.File.Pieces[pieceIndex].owners[msg.Sender.id] = msg.Sender
						}
						torrent.File.PiecesMutex.Unlock()


						torrent.PeerSwarm.interestingPeerMutex.Lock()
						_, isPresent := torrent.PeerSwarm.interestingPeer[msg.Sender.id]
						_, isPieceNeeded := torrent.File.neededPieceMap[pieceIndex]
						if !isPresent && isPieceNeeded{
							torrent.PeerSwarm.interestingPeer[msg.Sender.id] = msg.Sender
							requestMsg := GetMsg(MSG{MsgID: InterestedMsg})
							torrent.AddRequest(Request{msg: requestMsg, Peer: msg.Sender, Priority: 2})

							requestMsg2 := GetMsg(MSG{MsgID: UnchockeMsg})
							torrent.AddRequest(Request{msg: requestMsg2, Peer: msg.Sender, Priority: 3})
						}
						torrent.PeerSwarm.interestingPeerMutex.Unlock()
					}
					pieceIndex++
					bitIndex--
				}

			}
		}
	case InterestedMsg:
		msg.Sender.updateState(msg.Sender.chocked,true,torrent)
	case UnchockeMsg:
		msg.Sender.peerChocking = false
	case ChockeMsg:
		msg.Sender.peerChocking = true
	case PieceMsg:
	}

}

func (torrent *Torrent) saveTorrent() {
}

// assembles a complete piece
// writes piece to the file once completed
func (torrent *Torrent) addSubPiece(msg MSG, peer *Peer) error {
	var err error = nil
	subPieceIndex := int(math.Ceil(float64(msg.BeginIndex/SubPieceLen))) - 1

	if torrent.File.Pieces[msg.PieceIndex].Status != "empty" {
		torrent.File.Pieces[msg.PieceIndex].PieceTotalLen = int(msg.PieceLen)
		if torrent.File.Pieces[msg.PieceIndex].PieceIndex != int(msg.PieceIndex) {
			err = errors.New("wrong sub piece")
		}
	}

	if err != nil {
		return err
	}
	torrent.File.Pieces[msg.PieceIndex].CurrentLen += int(msg.PieceLen)

	torrent.File.Pieces[msg.PieceIndex].SubPieces[subPieceIndex] = make([]byte,len(msg.Piece))

	copy(torrent.File.Pieces[msg.PieceIndex].SubPieces[subPieceIndex], msg.Piece)

	// piece is complete add it to the file
	if torrent.File.Pieces[msg.PieceIndex].CurrentLen == torrent.File.Pieces[msg.PieceIndex].PieceTotalLen {
		torrent.File.Pieces[msg.PieceIndex].Status = "complete"
		file, _ := os.OpenFile(utils.DawnTorrentHomeDir+"/"+torrent.torrentMetaInfo.MapDict["info"].MapString["name"], os.O_CREATE|os.O_RDWR, os.ModePerm)

		delete(torrent.File.neededPieceMap, int(msg.PieceIndex))

		completePiece := bytes.Join(torrent.File.Pieces[msg.PieceIndex].SubPieces, []byte{})
		_, _ = file.WriteAt(completePiece, int64(msg.PieceIndex)*int64(torrent.File.PieceLen))

		// when a piece is complete, we update the index list
		sort.Sort(torrent.File)
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

func (torrent *Torrent) AddRequest(request Request) {
	fmt.Printf("adding request %v\n",request)
	torrent.requestQueueMutex.Lock()
	torrent.RequestQueue.Push(request)
	if atomic.LoadInt32(torrent.requestQueueAtomic) > 0{
			atomic.AddInt32(torrent.requestQueueAtomic,-atomic.LoadInt32(torrent.requestQueueAtomic))
				torrent.RequestQueueBlockMutex.Unlock()

	}
	torrent.requestQueueMutex.Unlock()

}

type Request struct {
	msg      []byte
	Peer     *Peer
	Priority int
}

func requestQueueComparator(a, b interface{}) int {
	n1 := a.(Request)
	n2 := b.(Request)

	switch {
	case n1.Priority < n2.Priority:
		return -1
	case n1.Priority > n2.Priority:
		return 1
	default:
		return 0
	}
}

//loop through the request priority queue, send request and block until new request are available
func (torrent *Torrent) RequestQueueManager() {
	i := 0
	ok := true
	var req interface{}

	for ok {
		torrent.requestQueueMutex.Lock()
		req, ok = torrent.RequestQueue.Pop()
		torrent.requestQueueMutex.Unlock()

		fmt.Printf("requestQueueManager %v\n", i)
		i++
		if ok {
			reqStruct := req.(Request)
			n, writeErr := reqStruct.Peer.connection.Write(reqStruct.msg)
			fmt.Printf("requestQueueManager wrote  %v bytes to %v with %v err\n", n, reqStruct.Peer.connection.RemoteAddr().String(), writeErr)
		} else {

			if torrent.RequestQueue.Size() == 0{
				fmt.Printf("requestQueueManager Locked\n")
				atomic.AddInt32(torrent.requestQueueAtomic,1)
				torrent.RequestQueueBlockMutex.Lock()
				fmt.Printf("requestQueueManager Unlocked\n")
			}

			ok = true
		}

	}
	os.Exit(10)
}

func (torrent *Torrent)worker(){
	requestWorker := 15
	for i := 0; i < requestWorker ;i++ {
		println("launch worker ",i)
		go torrent.RequestQueueManager()
	}

}

