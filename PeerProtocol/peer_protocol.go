package PeerProtocol

import (
	"DawnTorrent/JobQueue"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type Torrent struct {
	PeerSwarm  *PeerSwarm
	Downloader *TorrentDownloader

	jobQueue *JobQueue.WorkerPool

	PieceCounter     int
	chokeCounter     int
	done             chan bool
	LifeCycleChannel chan int

	PieceRequestManagerPeriodicFunc   *periodicFunc
	trackerRegularRequestPeriodicFunc *periodicFunc
	incomingConnectionPeriodicFunc    *periodicFunc
}

func NewTorrent(torrentPath string, mode int) *Torrent {
	torrent := new(Torrent)
	torrent.LifeCycleChannel = make(chan int)

	/// initializes or resumes a torrent a file
	fmt.Printf("mode %v", mode)
	if mode == InitTorrentFile_ {
		// creates a brand new Torrent
		torrent.Downloader = initDownloader(torrentPath)
	} else if mode == ResumeTorrentFile_ {
		// resumes a saved stopped torrent
		torrent.Downloader = resumeDownloader(torrentPath)
	}
	torrent.Downloader.torrent = torrent

	// initializes a peerSwarm (will manage peer connection and tracker request)
	torrent.PeerSwarm = NewPeerSwarm(torrent)

	torrent.PieceRequestManagerPeriodicFunc = newPeriodicFunc(torrent.Downloader.PieceRequestManager)
	torrent.trackerRegularRequestPeriodicFunc = newPeriodicFunc(torrent.PeerSwarm.trackerRegularRequest)
	torrent.jobQueue = JobQueue.NewWorkerPool(torrent, 15)

	torrent.jobQueue.Start()
	return torrent
}

func (torrent *Torrent) Pause() {
	//	fmt.Printf("\n file test %v\n", torrent.Downloader.InfoHashHex)
	torrent.Downloader.PiecesMutex.Lock()
	currentPiece := torrent.Downloader.Pieces[torrent.Downloader.CurrentPieceIndex]
	//	fmt.Printf("before len of neededSUb :%v\n", len(currentPiece.neededSubPiece))

	for _, req := range currentPiece.pendingRequest {
		req.status = nonStartedRequest
		currentPiece.neededSubPiece = append(currentPiece.neededSubPiece, req)
	}

	currentPiece.pendingRequest = make([]*PieceRequest, 0)
	//	torrent.Downloader.saveTorrent()

	torrent.PeerSwarm.killSwarm()

	//	fmt.Printf("len of neededSUb :%v\n", len(currentPiece.neededSubPiece))
	torrent.Downloader.PiecesMutex.Unlock()

}

func (torrent *Torrent) LifeCycle() {
	for {
		println("running......")

		 action := <-torrent.LifeCycleChannel
			switch action {
			case StartedState:
				if atomic.LoadInt32(torrent.Downloader.State) != StartedState {
					torrent.Downloader.SetState(StartedState)
					torrent.PieceRequestManagerPeriodicFunc.run()
					torrent.trackerRegularRequestPeriodicFunc.run()
					torrent.PieceRequestManagerPeriodicFunc.startFunc()
					torrent.trackerRegularRequestPeriodicFunc.startFunc()
					// will ignore the time interval limitation and send a request to the tracker as soon as possible
					//	torrent.trackerRegularRequestPeriodicFunc.byPassInterval = true
					torrent.PeerSwarm.peerOperation <- &peerOperation{operation: startReceivingIncomingConnection, peer: nil, freePeerChannel: nil}
				}
			case StoppedState:
				if atomic.LoadInt32(torrent.Downloader.State) == StartedState {
					torrent.Downloader.SetState(StoppedState)
					torrent.PieceRequestManagerPeriodicFunc.stopFunc()
					torrent.trackerRegularRequestPeriodicFunc.stopFunc()
					// will ignore the time interval limitation and send a request to the tracker as soon as possible
					//	torrent.trackerRegularRequestPeriodicFunc.byPassInterval = true
					torrent.PeerSwarm.peerOperation <- &peerOperation{operation: stopReceivingIncomingConnection, peer: nil, freePeerChannel: nil}
					//torrent.Pause()
					fmt.Printf("paused")
				}
			case sendTrackerRequest:
				// will ignore the time interval limitation and send a request to the tracker as soon as possible
				torrent.trackerRegularRequestPeriodicFunc.byPassInterval = true
			}
	}
}

//	Receives msg from other peers and calls the appropriate methods
func (torrent *Torrent) msgRouter(msg *MSG) {

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

func (torrent *Torrent) Worker(id int) {

	for request := range torrent.jobQueue.JobQueue {
		switch item := request.(type) {

		case *MSG:

			switch item.msgType {
			case outgoingMsg:
				torrent.PeerSwarm.peerMutex.Lock()
				defer func() {
					r := recover()
					if recover() != nil {
						println(r)
					}
				}()
				if item.Peer != nil {
					_, _ = item.Peer.connection.Write(item.RawMsg)
				}
				torrent.PeerSwarm.peerMutex.Unlock()

			case incomingMsg:
				torrent.msgRouter(item)
			}

		}
	}
}

type periodicFuncExecutor func(periodic *periodicFunc)

type periodicFunc struct {
	executor          periodicFuncExecutor
	byPassInterval    bool
	isRunning         bool
	stop              chan bool
	event             chan int
	interval          time.Duration
	lastExecTimeStamp time.Time
	isStop            bool
	tick              *time.Ticker
	wg                *sync.WaitGroup
}

func newPeriodicFunc(executor periodicFuncExecutor) *periodicFunc {
	periodicFunc := new(periodicFunc)
	periodicFunc.stop = make(chan bool)
	periodicFunc.interval = time.Millisecond * 200
	periodicFunc.tick = time.NewTicker(periodicFunc.interval)
	periodicFunc.executor = executor
	periodicFunc.lastExecTimeStamp = time.Now()

	return periodicFunc
}

func (periodicFunc *periodicFunc) run() {
	if !periodicFunc.isRunning {

		println("periodicFunc is running..........")
		go func() {
			for {
				select {
				case stop := <-periodicFunc.stop:
					periodicFunc.isStop = stop
					if !periodicFunc.isStop {
						periodicFunc.executor(periodicFunc)
					} else {

					}
				case <-periodicFunc.tick.C:
					//println("exec func ..............")
					if !periodicFunc.isStop {
						periodicFunc.executor(periodicFunc)
					}
				}

			}
		}()
	}
	periodicFunc.isRunning = true

}

func (periodicFunc *periodicFunc) startFunc() {
	periodicFunc.stop <- false
}

func (periodicFunc *periodicFunc) stopFunc() {
	periodicFunc.stop <- true
}
