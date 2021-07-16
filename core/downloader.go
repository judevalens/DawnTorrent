package core

import (
	"DawnTorrent/core/torrent"
	"DawnTorrent/interfaces"
	"DawnTorrent/rpc/torrent_state"
	"container/heap"
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	notStarted = iota
	inProgress = iota
	completed  = iota
)

const (
	random      = iota
	rarestFirst = iota
	endGame     = iota
	sequential  = iota
)

const (

)

type downloaderState interface {
}

type (
	updatePiecePriority func(i int)
	putPiece            func(msg PieceMsg)
	selectPiece         func() (*Piece, error)
)

/*
	Handles requesting and assembling pieces
*/
type Downloader struct {
	queue                  *downloadQueue
	Pieces                 []*Piece
	torrent                *torrent.Torrent
	peerManager            *PeerManager
	mode                   int
	selectedPiece          *Piece
	bitfield               []byte
	filesMetaData          []fileMetadata
	downloadJobs           chan *Piece
	peerChan               chan *PeerRequest
	signalChan             chan int
	peerSignalChan         chan int
	selectionChan          chan *Piece
	nWorker                int
	lastSelectedPieceIndex int
	totalSelectedPiece     int
	nPiece                 int
	stateChan              chan int
	writerChan             chan *Piece
	downloaded 			   int
}

func NewTorrentDownloader(torrent *torrent.Torrent, peerManager *PeerManager, stateChan chan int) *Downloader {

	downloader := new(Downloader)
	downloader.torrent = torrent
	downloader.peerManager = peerManager

	downloader.buildQueue()
	downloader.nWorker = 10
	downloader.downloadJobs = make(chan *Piece, downloader.nWorker)
	downloader.signalChan = make(chan int)
	downloader.selectionChan = make(chan *Piece)
	downloader.writerChan = make(chan *Piece)
	downloader.mode = sequential
	downloader.stateChan = stateChan
	downloader.totalSelectedPiece = 0
	return downloader
}

/*
	creates all the piece struct and push them to a priority queue so they can selected one by one
*/
func (downloader *Downloader) buildQueue() {
	downloader.nPiece = int(math.Ceil(float64(downloader.torrent.Length) / float64(downloader.torrent.PieceLength)))

	downloader.queue = new(downloadQueue)
	downloader.queue.mutex = &sync.Mutex{}
	log.Printf(":) file length %v, pieceLength %v, n pieces %v", downloader.torrent.Length, downloader.torrent.PieceLength, downloader.nPiece)

	downloader.Pieces = make([]*Piece, downloader.nPiece)
	for i := range downloader.Pieces {
		pieceLength := downloader.torrent.PieceLength

		startIndex := i * downloader.torrent.PieceLength
		currentTotalLength := startIndex + pieceLength

		if currentTotalLength > downloader.torrent.Length {
			pieceLength = downloader.torrent.Length - startIndex
		}

		downloader.Pieces[i] = NewPiece(i, downloader.torrent.PieceLength, pieceLength, notStarted)

		downloader.queue.Push(downloader.Pieces[i])
	}
	heap.Init(downloader.queue)

	bitfieldLen := int(math.Ceil(float64(downloader.nPiece) / 8))

	log.Debugf("#pieces %v, bitfieldLen %v", len(downloader.Pieces), bitfieldLen)
	downloader.bitfield = make([]byte, bitfieldLen)

}

/*
	makes the necessary initializations to Start or resume downloading a torrent.
*/
func (downloader *Downloader) Start(ctx context.Context,mainRoutineWaitG *sync.WaitGroup) {

	// selects a before calling the download()
	workerWaitGroup := &sync.WaitGroup{}
	ctx2,cancelFunc := context.WithCancel(context.TODO())

	// channels have to be reset

	downloadJobChan := make(chan *Piece, downloader.nWorker)
	signalChan := make(chan int)

	go downloader.download(ctx, workerWaitGroup,mainRoutineWaitG,cancelFunc,downloadJobChan,signalChan)
	// spawning workers
	go downloader.startWorker(ctx, workerWaitGroup,downloadJobChan,signalChan)

	// dependent routines
	go downloader.write(ctx2)
	go downloader.peerManager.GetAvailablePeer(ctx2)

}

/*
	Sends block request to peers. and resends block requests that have not been fulfilled by peers
*/

func (downloader *Downloader) startWorker(ctx context.Context, workerWaitGroup *sync.WaitGroup,downloadJobChan chan *Piece,signalChan chan int) {
	i := 0
	for i < downloader.nWorker {
		workerCtx,_ := context.WithCancel(ctx)
		go func(ctx2 context.Context, workerWaitGroup *sync.WaitGroup, i int,downloadJobChan chan *Piece,signalChan chan int) {
			workerWaitGroup.Add(1)
			log.Printf("starting worker %v",i)
			signalChan <- i
			log.Printf("worker %v sent initial signal", i)
			for {
				select {
				case <-ctx2.Done():
					log.Printf("shutting worker %v down 1", i)
					workerWaitGroup.Done()
					return
				case piece := <-downloadJobChan:
					if piece == nil {
						log.Printf("shutting worker %v down 2", i)
						workerWaitGroup.Done()
						return
					}

					//time.Sleep(time.Second)
					peerRequest := &PeerRequest{
						pieceIndex: piece.PieceIndex,
						response:   make(chan *Peer),
					}

					log.Infof("worker %v requesting piece %v ...., length %v",i,piece.PieceIndex,piece.pieceLength)

					downloader.peerManager.peerChan <- peerRequest
					peer := <-peerRequest.response
					close(peerRequest.response)
					log.Debugf("receive peer: id %v", peer.id)
					err := piece.download(peer)
					if err != nil {
						//TODO need to be handled properly
						log.Panicf("worker %v failed\n%v",i,err.Error())
						downloader.queue.Push(piece)
						return
					}
					downloader.writerChan <- piece

					atomic.AddInt64(&peer.IsFree, -1)

					downloader.peerManager.peerAlert.L.Lock()
					downloader.peerManager.peerAlert.Signal()
					downloader.peerManager.peerAlert.L.Unlock()

					log.Debugf("worker %v, just downloaded a piece : %v", i, piece.PieceIndex)

					signalChan <- i
				}

			}

		}(workerCtx, workerWaitGroup, i,downloadJobChan,signalChan)
		i++
	}


}

func (downloader *Downloader) download(ctx context.Context, workerWaitGroup *sync.WaitGroup,mainRoutinesWg *sync.WaitGroup,cancelChildRoutines context.CancelFunc,downloadJobChan chan *Piece,signalChan chan int) {

	sentShutDownSignal := false
	isShuttingDown := false
	localCtx,cancelFunc := context.WithCancel(context.TODO())
	for {
		select {
		case <-ctx.Done():
			isShuttingDown = true
		case i := <-signalChan:

			piece, isCompleted, err := downloader.selectPiece()
			if err != nil{
				log.Panicf("failed to select piece\n%v",err.Error())
			}
			if isCompleted||isShuttingDown{

				if !sentShutDownSignal {
					go func() {
						workerWaitGroup.Wait()
						cancelChildRoutines()
						mainRoutinesWg.Done()
						cancelFunc()

						if isCompleted {
							downloader.stateChan <- interfaces.CompleteTorrent
						}
					}()
					sentShutDownSignal = true
				}
			}

			log.Debugf("received signal from worker %v", i)
			downloadJobChan <- piece
		case <- localCtx.Done():
			println("actual end")
			return

		}

	}
}


func (downloader *Downloader) putBlock(msg PieceMsg) {
	downloader.Pieces[msg.PieceIndex].putPiece(msg)
}

/*
	updates a piece's availability when a bitmask or have msg is received
*/
func (downloader *Downloader) updatePiecePriority(peer *Peer) {
	log.Debugf("updating piece priority")
	for _, piece := range downloader.Pieces {
		piece.updateAvailability(incrementAvailability, peer)
		if piece.QueueIndex > -1 {
			downloader.queue.fixQueue(piece.QueueIndex)
		}
	}
}

/*
	selects the next piece to be downloaded. piece can be selected randomly or by availability
*/
func (downloader *Downloader) selectPiece() (*Piece, bool, error) {

	if downloader.totalSelectedPiece >= len(downloader.Pieces) {
		return nil, true, nil
	}

	var index int
	var selectedPiece *Piece
	if downloader.mode == random {
		randGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
		index = randGenerator.Intn(downloader.queue.Len())

	} else if downloader.mode == rarestFirst {
		selectedPiece = downloader.queue.Pop().(*Piece)
		if selectedPiece == nil {
			return nil,false, errors.New("pendingRequest is empty")
		}
		return selectedPiece,false, nil
	} else {
		downloader.selectedPiece = downloader.Pieces[downloader.lastSelectedPieceIndex]
		downloader.lastSelectedPieceIndex++
		downloader.totalSelectedPiece++

		return downloader.selectedPiece,false, nil
	}
	selectedPiece, err := downloader.queue.RemoveAt(index)
	downloader.selectedPiece = selectedPiece
	if err != nil {
		return nil,false, err
	}
	return selectedPiece,false, err
}

func (downloader *Downloader) write(ctx context.Context) {

	for {
		select {
		case <-ctx.Done():
			log.Infof("shutting down piece writer.....")
			return
		case piece := <-downloader.writerChan:
			err := writePiece(piece, downloader.torrent.FileSegments)
			if err != nil {
				log.Fatal(err)
				return
			}

		}
	}

}

func (downloader *Downloader) serialize() *torrent_state.Stats{
	return &torrent_state.Stats{
		Bitfield:      downloader.bitfield,
		TorrentLength: int32(downloader.torrent.Length),
		CurrentLength: 0,
		DownloadRate: 0,
		UploadRate: 0,
	}
}
