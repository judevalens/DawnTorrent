package app

import (
	"DawnTorrent/app/torrent"
	"DawnTorrent/interfaces"
	"container/heap"
	"context"
	"errors"
	"fmt"
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
	queueLength = 5
	BlockLen    = 16384
	maxReq      = 10
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
	queue            *downloadQueue
	Pieces           []*Piece
	torrent          *torrent.Torrent
	peerManager      *PeerManager
	mode             int
	selectedPiece    *Piece
	blockRequests    *sync.Map
	pendingRequestsC int64
	ticker           chan time.Time
	nPiece           int
	bitfield         []byte
	SyncOperation    chan interfaces.SyncOp
	mutex            *sync.Mutex
	writer           PieceWriter
	filesMetaData    []fileMetadata

	jobs           chan *Piece
	peerChan       chan *PeerRequest
	signalChan     chan int
	peerSignalChan chan int
	selectionChan 	chan *Piece
	nWorker        int
	lastSelectedPieceIndex int
	totalSelectedPiece int
	stateChan		chan int

}

func NewTorrentDownloader(torrent *torrent.Torrent, peerManager *PeerManager,stateChan chan int) *Downloader {

	downloader := new(Downloader)
	downloader.torrent = torrent
	downloader.peerManager = peerManager
	downloader.SyncOperation = make(chan interfaces.SyncOp)
	downloader.blockRequests = &sync.Map{}
	downloader.ticker = make(chan time.Time)
	downloader.mutex = new(sync.Mutex)
	downloader.buildQueue()
	downloader.nWorker = 15
	downloader.jobs = make(chan *Piece, downloader.nWorker)
	downloader.signalChan = make(chan int)
	downloader.selectionChan = make(chan *Piece)
	downloader.writer = PieceWriter{
		segments:    downloader.torrent.FileSegments,
		pieceLength: torrent.PieceLength,
	}
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
func (downloader *Downloader) Start(ctx context.Context) {

	// selects a before calling the download()
	workerWaitGroup :=  &sync.WaitGroup{}

	go downloader.download(ctx,workerWaitGroup)
	go downloader.startWorker(ctx,workerWaitGroup)

}

/*
	Sends block request to peers. and resends block requests that have not been fulfilled by peers
*/

func (downloader *Downloader) startWorker(ctx context.Context,workerWaitGroup *sync.WaitGroup) {
	i := 0
	for i < downloader.nWorker {
		go func(ctx2 context.Context, workerWaitGroup *sync.WaitGroup, i int) {
			workerWaitGroup.Add(1)
			for {
				select {
				case <-ctx2.Done():
					//log.Printf("shutting worker %v down",i)
					return
				case piece := <-downloader.jobs:
					if piece == nil{
						log.Printf("shutting worker %v down 2",i)
						workerWaitGroup.Done()
						return
					}
					peerRequest := &PeerRequest{
						pieceIndex: piece.PieceIndex,
						response: make(chan *Peer),
					}

					log.Debug("requesting piece....")


					downloader.peerManager.peerChan <- peerRequest
					peer := <- peerRequest.response
					close(peerRequest.response)
					log.Debugf("receive peer: id %v",peer.id)
					err := piece.download(peer)
					if err != nil {
						log.Fatal(err)
						return 
					}
					fmt.Printf("peer %v\n",atomic.LoadInt64(&peer.IsFree))
					atomic.AddInt64(&peer.IsFree,-1)

					downloader.peerManager.peerAlert.L.Lock()
					downloader.peerManager.peerAlert.Signal()
					downloader.peerManager.peerAlert.L.Unlock()


					log.Debugf("worker %v, just downloaded a piece : %v", i,piece.PieceIndex)

					downloader.signalChan <- i
				}

			}

		}(ctx, workerWaitGroup,i)
		i++
	}
	i = 0

	for i < downloader.nWorker {

		go func(workerIndex int) {
			downloader.signalChan <- workerIndex
		}(i)

		i++
	}

}

func (downloader *Downloader) download(ctx context.Context, workerWaitGroup *sync.WaitGroup) {

	sentShutDownSignal := false

	for {
		select {
		case <-ctx.Done():
			return
		case i := <-downloader.signalChan:
			piece, err := downloader.selectPiece()
			if err != nil{

				if !sentShutDownSignal{
					go func() {
						workerWaitGroup.Wait()
						downloader.stateChan <- interfaces.CompleteTorrent
					}()
					sentShutDownSignal = true
				}

				//downloader.stateChan <- interfaces.CompleteTorrent
				//close(downloader.jobs)
				//time.Sleep(time.Second*9282)
			}
			log.Debugf("received signal from worker %v", i)
			downloader.jobs <- piece

		}

	}
}

func (downloader *Downloader) stopDownloading() {
	close(downloader.jobs)
}

func (downloader *Downloader) putBlock(msg PieceMsg){
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
func (downloader *Downloader) selectPiece() (*Piece, error) {

	if downloader.totalSelectedPiece >= len(downloader.Pieces){
		return nil,errors.New("queue is empty")
	}

	var index int
	var selectedPiece *Piece
	if downloader.mode == random {
		randGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
		index = randGenerator.Intn(downloader.queue.Len())

	} else if downloader.mode == rarestFirst {
		downloader.mutex.Lock()
		selectedPiece = downloader.queue.Pop().(*Piece)
		downloader.mutex.Unlock()
		if selectedPiece == nil {
			return nil, errors.New("pendingRequest is empty")
		}
		return selectedPiece, nil
	} else {

		log.Printf("queue len %v", downloader.totalSelectedPiece)
		downloader.selectedPiece = downloader.Pieces[downloader.lastSelectedPieceIndex]
		downloader.lastSelectedPieceIndex++
		downloader.totalSelectedPiece++

		return downloader.selectedPiece, nil
	}
	selectedPiece, err := downloader.queue.RemoveAt(index, downloader.mutex)
	downloader.selectedPiece = selectedPiece
	if err != nil {
		return nil, err
	}
	return selectedPiece, err
}


func (downloader *Downloader) write(ctx context.Context){


}
