package app

import (
	"DawnTorrent/interfaces"
	"container/heap"
	"context"
	"errors"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
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
)

const (
	queueLength    = 5
	BlockLen       = 1024
	maxReq         = 10
	requestTimeOut = time.Second * 2
)

type downloaderState interface {
}

type (
	updatePiecePriority func(i int)
	putPiece            func(msg PieceMsg)
	selectPiece			func() (*Piece, error)
)

/*
	Handles requesting and assembling pieces
 */
type Downloader struct {
	queue          *downloadQueue
	Pieces         []*Piece
	torrent        *Torrent
	torrentManager *TorrentManager
	peerManager    *PeerManager
	mode           int
	selectedPiece  *Piece
	pendingRequest map[string]*BlockRequest
	ticker         chan time.Time
	nPiece         int
	bitfield       []byte
	SyncOperation  chan interfaces.SyncOp
	mutex          *sync.Mutex
	writer   	PieceWriter
}

func NewTorrentDownloader(torrent *Torrent,manager *TorrentManager,peerManager *PeerManager) *Downloader {
	downloader := new(Downloader)
	downloader.torrent = torrent
	downloader.torrentManager = manager
	downloader.peerManager = peerManager
	downloader.SyncOperation = make(chan interfaces.SyncOp)
	downloader.pendingRequest =  make(map[string]*BlockRequest)
	downloader.ticker = make(chan time.Time)
	downloader.mutex = new(sync.Mutex)
	downloader.buildQueue()
	downloader.writer = PieceWriter{
		metaData: downloader.torrent.FilesMetadata,
		pieceLength: torrent.pieceLength,
	}
	return downloader
}
/*
	creates all the piece struct and push them to a priority queue so they can selected one by one
*/
func (downloader *Downloader) buildQueue() {
	downloader.nPiece = int(math.Ceil(float64(downloader.torrent.FileLength) / float64(downloader.torrent.pieceLength)))
	downloader.queue = new(downloadQueue)

	log.Printf("file length %v, pieceLength %v, n pieces %v", downloader.torrent.FileLength, downloader.torrent.pieceLength, downloader.torrent.nPiece)

	downloader.Pieces = make([]*Piece, downloader.nPiece)
	for i, _ := range downloader.Pieces {
		pieceLen := downloader.torrent.pieceLength

		startIndex := i * downloader.torrent.pieceLength
		currentTotalLength := startIndex + pieceLen

		if currentTotalLength > downloader.torrent.FileLength {
			pieceLen = currentTotalLength % downloader.torrent.FileLength
		}

		downloader.Pieces[i] = NewPiece(i, downloader.torrent.pieceLength, pieceLen, notStarted)
		downloader.queue.Push(downloader.Pieces[i])
	}
	heap.Init(downloader.queue)

	log.Printf("queue length %v ", len(*downloader.queue))


	bitfieldLen := int(math.Ceil(float64(downloader.nPiece) / 8))

	log.Printf("bitfield len %v",bitfieldLen)
	downloader.bitfield = make([]byte,bitfieldLen)

}


/*
	makes the necessary initializations to Start or resume downloading a torrent.
*/
func (downloader *Downloader) Start(ctx context.Context) {

	// selects a before calling the download()
	downloader.torrentManager.SyncOperation <- func() {

		piece, err := downloader.selectPiece()
		if err != nil {
			return
		}

		downloader.selectedPiece = piece
	}

	// sends a signal through the ticker channel to fire download()
	go func() {
		downloader.ticker <- time.Now()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-downloader.ticker:
			downloader.download(ctx)
			// this basically calls the download method after a certain amount of time
			time.AfterFunc(time.Millisecond*150, func() {
				downloader.ticker <- time.Now()
			})

		}
	}
}

/*
	Sends block request to peers. and resends block requests that have not been fulfilled by peers
 */
func (downloader *Downloader) download(ctx context.Context) {

	var err error



	for _, request := range downloader.pendingRequest {

		if time.Now().Sub(request.TimeStamp) > requestTimeOut {

			peer, err := downloader.peerManager.GetAvailablePeer()

			downloader.sendRequest(peer, request)
			os.Exit(100)

			if err != nil {
				return
			}

		}
	}

	// sending fresh requests
	availablePeer, err := downloader.peerManager.GetAvailablePeer()

	for err != nil {
		requests := downloader.selectedPiece.getNextRequest(10)

		downloader.sendRequest(availablePeer, requests...)
		availablePeer, err = downloader.peerManager.GetAvailablePeer()
	}

}

/*
	Sends  pieceRequests to a Peer
 */
func (downloader *Downloader) sendRequest(peer *Peer, pieceRequest ...*BlockRequest) {

	for i, request := range pieceRequest {
		_, err := peer.SendMsg(request.Marshal())
		if err != nil {
			return
		}
		// each request must be completed in at least requestTimeOut
		request.TimeStamp = time.Now().Add(requestTimeOut * time.Duration(int64(i)*time.Nanosecond.Nanoseconds()))
		request.Providers = append(request.Providers, peer)
		downloader.pendingRequest[request.Id] = request
	}

}
/*
	updates a piece's availability when a bitmask or have msg is received
 */
func  (downloader *Downloader) updatePiecePriority(peer *Peer) {
	log.Print("updating piece priority")
	for _, piece := range downloader.Pieces {

		piece.updateAvailability(1, peer)
		if piece.QueueIndex > -1 {
			downloader.queue.fixQueue(piece.QueueIndex)
		}
	}
}
/*
	selects the next piece to be downloaded. piece can be selected randomly or by availability
 */
func (downloader *Downloader) selectPiece() (*Piece, error) {
	if downloader.selectedPiece != nil {
		return downloader.selectedPiece, nil
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
	}
	selectedPiece, err := downloader.queue.RemoveAt(index,downloader.mutex)

	downloader.selectedPiece = selectedPiece
	if err != nil {
		return nil, err
	}
	return selectedPiece, err

}


func (downloader *Downloader) PutPiece(msg PieceMsg) {

	if !downloader.selectedPiece.hasSubPiece(msg.BeginIndex) {
		downloader.selectedPiece.UpdateBitfield(msg.BeginIndex)

		isComplete := downloader.writer.writeToBuffer(msg.Payload, msg.BeginIndex)

		if isComplete {

			err := downloader.writer.writePiece()
			if err != nil {
				return
			}

			downloader.torrentManager.SyncOperation <- func() {

				piece, err := downloader.selectPiece()
				if err != nil {
					return 
				}

				downloader.selectedPiece = piece
			}

		}
	}

}

func (downloader *Downloader) resolveRequest(piece RequestMsg) {

	pieceRequestId := getRequestId(piece.PieceIndex, piece.BeginIndex)
	pieceRequest := downloader.pendingRequest[pieceRequestId]
	pieceRequest.FullFilled = true

	for _, provider := range pieceRequest.Providers {
		delete(provider.PendingRequest, pieceRequestId)
	}
}

