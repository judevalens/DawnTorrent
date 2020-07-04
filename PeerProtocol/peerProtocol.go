package PeerProtocol

import (
	"DawnTorrent/JobQueue"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	pieceHashLen = 20
)

type Torrent struct {
	PeerSwarm        *PeerSwarm
	File             *TorrentFile
	requestQueue     *JobQueue.Queue
	incomingMsgQueue *JobQueue.Queue
	pieceQueue       *JobQueue.Queue

	jobQueue *JobQueue.WorkerPool

	PieceCounter int
	chokeCounter int
	done         chan bool
	lifeCycleWG  *sync.WaitGroup
	StartedState        chan int
	StoppedState chan int


	stopPieceRequestLoop chan bool
	stopTrackerLoop chan bool


	PieceRequestManagerPeriodicFunc  *periodicFunc
	trackerRegularRequestPeriodicFunc *periodicFunc
}

func NewTorrent(torrentPath string, mode int) *Torrent {
	torrent := new(Torrent)
	torrent.StartedState = make(chan int)
	torrent.StoppedState = make(chan int)
	torrent.stopPieceRequestLoop = make(chan bool)
	torrent.stopTrackerLoop = make(chan bool)
	torrent.lifeCycleWG  = new(sync.WaitGroup)



	fmt.Printf("mode %v",mode)
	if mode == InitTorrentFile_ {
		// creates a brand new Torrent
		torrent.File = InitTorrentFile(torrent, torrentPath)
	} else if mode == ResumeTorrentFile_ {
		// resumes a saved stopped torrent
		torrent.File = resumeTorrentFile(torrentPath)
	}
	torrent.PeerSwarm = NewPeerSwarm(torrent)
	//torrent.requestQueue = JobQueue.NewQueue(torrent)
	//torrent.incomingMsgQueue = JobQueue.NewQueue(torrent)
	//torrent.pieceQueue = JobQueue.NewQueue(torrent)
	//torrent.requestQueue.Run(1)
	//torrent.incomingMsgQueue.Run(1)
	//torrent.pieceQueue.Run(1)
	torrent.PieceRequestManagerPeriodicFunc = newPeriodicFunc(torrent.PieceRequestManager2)
	torrent.trackerRegularRequestPeriodicFunc = newPeriodicFunc(torrent.PeerSwarm.trackerRegularRequest)
	torrent.jobQueue = JobQueue.NewWorkerPool(torrent,15)

	torrent.jobQueue.Start()
	return torrent
}

func (torrent *Torrent) Pause() {
		fmt.Printf("\n file test %v\n", torrent.File.InfoHashHex)
	torrent.File.PiecesMutex.Lock()
		currentPiece := torrent.File.Pieces[torrent.File.CurrentPieceIndex]
		fmt.Printf("before len of neededSUb :%v\n", len(currentPiece.neededSubPiece))


		for _,req := range currentPiece.pendingRequest{
			req.status = nonStartedRequest
			currentPiece.neededSubPiece  = append(currentPiece.neededSubPiece,req)
		}

		currentPiece.pendingRequest = make([]*PieceRequest,0)
		torrent.File.saveTorrent()

		torrent.PeerSwarm.killSwarm()

		fmt.Printf("len of neededSUb :%v\n", len(currentPiece.neededSubPiece))
		torrent.File.PiecesMutex.Unlock()

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
		time.Sleep(time.Millisecond * 25)

	}

}

//	Adds a request for piece to the a job Queue
//	Params :
//	subPieceRequest *PieceRequest : contains the raw request msg
func (torrent *Torrent) requestPiece(subPieceRequest *PieceRequest) error {

	println("im not stuck! 1")
	torrent.PeerSwarm.SortPeerByDownloadRate()
	var err error

	peerOperation := new(peerOperation)
	peerOperation.operation = isPeerFree
	peerOperation.freePeerChannel = make(chan *Peer)
	currentPiece := torrent.File.Pieces[torrent.File.CurrentPieceIndex]
	var peer *Peer
	/*
	peerIndex := 0

	isPeerFree := false

	for !isPeerFree && peerIndex < torrent.PeerSwarm.PeerByDownloadRate.Size(){
		fmt.Printf("looking for a suitable peer | # of available peer : %v\n", torrent.PeerSwarm.PeerByDownloadRate.Size())
		//	fmt.Printf("\nchoking peer # %v\n",torrent.chokeCounter)
		fmt.Printf("looking for a suitable peer | # of available peer active connection: %v\n", torrent.PeerSwarm.activeConnection.Size())
		peerI, found := torrent.PeerSwarm.PeerByDownloadRate.Get(peerIndex)

		if found && peerI != nil {
			peer = peerI.(*Peer)
			peerIndex = peerIndex % torrent.PeerSwarm.PeerByDownloadRate.Size()

			if peer != nil {
				isPeerFree = peer.isPeerFree()

				//	fmt.Printf("peer is not free, # pendingg Req : %v | peer App : %v\n ",len(peer.peerPendingRequest),peerIndex)
			} else {
				//fmt.Printf("peer is nil ")

				isPeerFree = false
			}
		} else {
		}

	}
	*/
	torrent.PeerSwarm.peerOperation <- peerOperation
	peer = <- peerOperation.freePeerChannel


	if peer != nil {
		subPieceRequest.msg.Peer = peer
		torrent.jobQueue.AddJob(subPieceRequest.msg)
		peer.mutex.Lock()
		peer.peerPendingRequest = append(peer.peerPendingRequest, subPieceRequest)
		peer.mutex.Unlock()

		subPieceRequest.timeStamp = time.Now()

		if subPieceRequest.status == nonStartedRequest {
			subPieceRequest.status = pendingRequest
			currentPiece.pendingRequest = append(currentPiece.pendingRequest, subPieceRequest)
		}
		//fmt.Printf("request id %v %v \n",subPieceRequest.startIndex, subPieceRequest.msg)

	} else {
		err = errors.New("request is not sent ")
	}

	return err
}

//	Selects Pieces that need to be downloader
//	When a piece is completely downloaded , a new one is selected
func (torrent *Torrent) PieceRequestManager() {
	println("im not stuck! 2")

	torrent.File.timeS = time.Now()
	for atomic.LoadInt32(torrent.File.Status) == StartedState {

		torrent.File.PiecesMutex.Lock()
		currentPiece := torrent.File.Pieces[torrent.File.CurrentPieceIndex]
		torrent.File.SelectNewPiece = currentPiece.CurrentLen == currentPiece.Len


		if !torrent.File.SelectNewPiece && len(currentPiece.neededSubPiece) == 0 && len(currentPiece.pendingRequest) == 0{
			log.Printf("I messed up")
		}

		if torrent.File.SelectNewPiece {
			if torrent.File.PieceSelectionBehavior == "random" {
				torrent.File.CurrentPieceIndex++
				fmt.Printf("switching piece # %v\n", torrent.File.CurrentPieceIndex)

				torrent.File.SelectNewPiece = false
				torrent.PieceCounter = 0

			}
		}else{

			fmt.Printf("not complete yet , currenLen %v , actual Len %v\n",currentPiece.CurrentLen,currentPiece.Len)
		}

		currentPiece.pendingRequestMutex.Lock()
		nSubPieceRequest := len(currentPiece.pendingRequest)

		//	the number of subPiece request shall not exceed maxRequest
		//	new requests are sent if the previous request have been fulfilled
		if nSubPieceRequest < maxRequest {
			nReq := maxRequest - len(currentPiece.pendingRequest)

			nReq = int(math.Min(float64(nReq), float64(len(currentPiece.neededSubPiece))))

			i := 0
			randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

			//fmt.Printf("n pending Request : %v  nReq: %v needed Subp : %v\n",len(currentPiece.pendingRequest),nReq,len(currentPiece.neededSubPiece))

			for i < nReq {
				// 	subPieces request are randomly selected
				randomN := randomSeed.Intn(len(currentPiece.neededSubPiece))

				req := currentPiece.neededSubPiece[randomN]

				//fmt.Printf("sending request\n")
				err := torrent.requestPiece(req)

				// once a request is added to the job JobQueue, it is removed from the needed subPiece / prevents from being reselected
				if err == nil {
					currentPiece.neededSubPiece[randomN] = currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1]
					currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1] = nil
					currentPiece.neededSubPiece = currentPiece.neededSubPiece[:len(currentPiece.neededSubPiece)-1]
				}else{

					fmt.Printf("ERR : %v\n",err)
					//os.Exit(99)
				}
				i++
				//	fmt.Printf("sending new Request.....\n")

			}
			//fmt.Printf("sending new Request2.....\n")

		}

		// re-sends request that have been pending for a certain amounts of time
		nPendingRequest := len(currentPiece.pendingRequest)

		for r := 0; r < nPendingRequest; r++ {
			pendingRequest := currentPiece.pendingRequest[r]

			if time.Now().Sub(pendingRequest.timeStamp) >= maxWaitingTime {
				//fmt.Printf("sending request\n")
				_ = torrent.requestPiece(pendingRequest)
			}

			//fmt.Printf("Resending Request.....\n")

		}
		currentPiece.pendingRequestMutex.Unlock()

		torrent.File.PiecesMutex.Unlock()

		//	fmt.Printf("Resending Request 3.....\n")


		time.Sleep(time.Millisecond * 25)
	}

}
func (torrent *Torrent) PieceRequestManager2(periodic *periodicFunc) {
	if atomic.LoadInt32(torrent.File.Status) == StartedState {

		torrent.File.PiecesMutex.Lock()
		currentPiece := torrent.File.Pieces[torrent.File.CurrentPieceIndex]
		torrent.File.SelectNewPiece = currentPiece.CurrentLen == currentPiece.Len


		if !torrent.File.SelectNewPiece && len(currentPiece.neededSubPiece) == 0 && len(currentPiece.pendingRequest) == 0{
			log.Printf("I messed up")
		}

		if torrent.File.SelectNewPiece {
			if torrent.File.PieceSelectionBehavior == "random" {
				torrent.File.CurrentPieceIndex++
				fmt.Printf("switching piece # %v\n", torrent.File.CurrentPieceIndex)

				torrent.File.SelectNewPiece = false
				torrent.PieceCounter = 0

			}
		}else{

			fmt.Printf("not complete yet , currenLen %v , actual Len %v\n",currentPiece.CurrentLen,currentPiece.Len)
		}

		currentPiece.pendingRequestMutex.Lock()
		nSubPieceRequest := len(currentPiece.pendingRequest)

		//	the number of subPiece request shall not exceed maxRequest
		//	new requests are sent if the previous request have been fulfilled
		if nSubPieceRequest < maxRequest {
			nReq := maxRequest - len(currentPiece.pendingRequest)

			nReq = int(math.Min(float64(nReq), float64(len(currentPiece.neededSubPiece))))

			i := 0
			randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

			//fmt.Printf("n pending Request : %v  nReq: %v needed Subp : %v\n",len(currentPiece.pendingRequest),nReq,len(currentPiece.neededSubPiece))

			for i < nReq {
				// 	subPieces request are randomly selected
				randomN := randomSeed.Intn(len(currentPiece.neededSubPiece))

				req := currentPiece.neededSubPiece[randomN]

				//fmt.Printf("sending request\n")
				err := torrent.requestPiece(req)

				// once a request is added to the job JobQueue, it is removed from the needed subPiece / prevents from being reselected
				if err == nil {
					currentPiece.neededSubPiece[randomN] = currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1]
					currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1] = nil
					currentPiece.neededSubPiece = currentPiece.neededSubPiece[:len(currentPiece.neededSubPiece)-1]
				}else{

					fmt.Printf("ERR : %v\n",err)
					//os.Exit(99)
				}
				i++
				//	fmt.Printf("sending new Request.....\n")

			}
			//fmt.Printf("sending new Request2.....\n")

		}

		// re-sends request that have been pending for a certain amounts of time
		nPendingRequest := len(currentPiece.pendingRequest)

		for r := 0; r < nPendingRequest; r++ {
			pendingRequest := currentPiece.pendingRequest[r]

			if time.Now().Sub(pendingRequest.timeStamp) >= maxWaitingTime {
				//fmt.Printf("sending request\n")
				_ = torrent.requestPiece(pendingRequest)
			}

			//fmt.Printf("Resending Request.....\n")

		}
		currentPiece.pendingRequestMutex.Unlock()

		torrent.File.PiecesMutex.Unlock()

		//	fmt.Printf("Resending Request 3.....\n")
		}

}

func (torrent *Torrent) LifeCycle(){
	for {
		println("running......")
		select{
		case <- torrent.StartedState:
			if atomic.LoadInt32(torrent.File.Status) != StartedState {
				torrent.File.setState(StartedState)
				torrent.PieceRequestManagerPeriodicFunc.run()
				torrent.trackerRegularRequestPeriodicFunc.run()
				torrent.PieceRequestManagerPeriodicFunc.startFunc()
				torrent.trackerRegularRequestPeriodicFunc.startFunc()
			}
		case <- torrent.StoppedState:
			if atomic.LoadInt32(torrent.File.Status) == StartedState {
				torrent.File.setState(StoppedState)
				torrent.PieceRequestManagerPeriodicFunc.stopFunc()
				torrent.trackerRegularRequestPeriodicFunc.stopFunc()
				//torrent.Pause()
				fmt.Printf("paused")
			}

		}
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
		if msg.PieceIndex < torrent.File.nPiece {
			// verifies that the length of the data is not greater or smaller than amount requested
			if msg.PieceLen == torrent.File.subPieceLen || msg.PieceLen == torrent.File.Pieces[msg.PieceIndex].Len%torrent.File.subPieceLen {
			//	_ = torrent.File.AddSubPiece(msg, msg.Peer)

				torrent.File.addPieceChannel <- msg
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

/*func (torrent *Torrent) Worker(queue *JobQueue.Queue, id int) {
	for {
		request := queue.Pop()
		switch item := request.(type) {

		case *MSG:

			switch item.msgType {
			case outgoingMsg:
				torrent.PeerSwarm.peerMutex.Lock()
				defer func() {
					r := recover()
					if recover() != nil{
						println(r)
					}
				}()
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
*/
func (torrent *Torrent)Worker(id int){

	for request := range  torrent.jobQueue.JobQueue {
		switch item := request.(type) {

		case *MSG:

			switch item.msgType {
			case outgoingMsg:
				torrent.PeerSwarm.peerMutex.Lock()
				defer func() {
					r := recover()
					if recover() != nil{
						println(r)
					}
				}()
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


type periodicFuncExecutor  func(periodic *periodicFunc)

type periodicFunc struct {
	executor periodicFuncExecutor
	isRunning bool
	stop chan bool
	interval time.Duration
	lastExecTimeStamp time.Time
	isStop bool
	tick *time.Ticker
	wg *sync.WaitGroup
}

func newPeriodicFunc(executor periodicFuncExecutor) *periodicFunc{
	periodicFunc := new(periodicFunc)
	periodicFunc.stop = make(chan bool)
	periodicFunc.interval = time.Millisecond*25
	periodicFunc.tick = time.NewTicker(periodicFunc.interval)
	periodicFunc.executor = executor
	periodicFunc.lastExecTimeStamp = time.Now()

	return periodicFunc
}

func(periodicFunc *periodicFunc) run(){
	if !periodicFunc.isRunning{

		println("periodicFunc is running..........")
		go func() {
			for{
				select {

				case stop := <- periodicFunc.stop:
					periodicFunc.isStop = stop
					if !periodicFunc.isStop {
						periodicFunc.executor(periodicFunc)
					}
				case  <- periodicFunc.tick.C :
					println("exec func ..............")
					if !periodicFunc.isStop{
						periodicFunc.executor(periodicFunc)
					}
				}

			}
		}()
	}
	periodicFunc.isRunning = true

}

func (periodicFunc *periodicFunc) startFunc(){
	periodicFunc.stop <- false
}

func (periodicFunc *periodicFunc) stopFunc(){
	periodicFunc.stop <- true
}