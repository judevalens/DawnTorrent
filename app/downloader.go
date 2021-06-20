package app

import (
	"DawnTorrent/interfaces"
	"container/heap"
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"math"
	"math/rand"
	"os"
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
	writer         PieceWriter
}

func NewTorrentDownloader(torrent *Torrent, manager *TorrentManager, peerManager *PeerManager) *Downloader {

	downloader := new(Downloader)
	downloader.torrent = torrent
	downloader.torrentManager = manager
	downloader.peerManager = peerManager
	downloader.SyncOperation = make(chan interfaces.SyncOp)
	downloader.pendingRequest = make(map[string]*BlockRequest)
	downloader.ticker = make(chan time.Time)
	downloader.mutex = new(sync.Mutex)
	downloader.buildQueue()
	downloader.writer = PieceWriter{
		metaData:    downloader.torrent.FilesMetadata,
		pieceLength: torrent.pieceLength,
	}
	downloader.mode = sequential
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
	for i := range downloader.Pieces {
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


	bitfieldLen := int(math.Ceil(float64(downloader.nPiece) / 8))

	downloader.bitfield = make([]byte, bitfieldLen)

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

		downloader.writer.setNewPiece(downloader.selectedPiece.pieceLength, downloader.selectedPiece.PieceIndex)
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

			time.AfterFunc(time.Millisecond*250, func() {
				downloader.ticker <- time.Now()
			})

			//time.Sleep(time.Millisecond*2000)
			// this basically calls the download method after a certain amount of time

		}
	}
}

/*
	Sends block request to peers. and resends block requests that have not been fulfilled by peers
*/

func (downloader *Downloader) download(ctx context.Context) {
	var err error
	var peer *Peer

	for _, request := range downloader.pendingRequest {

		if request.timedOut() || request.hasNoProvider() {

			if peer == nil || !peer.isAvailable(request.Id) {
				peer, err = downloader.peerManager.GetAvailablePeer(request.Id,downloader.selectedPiece.PieceIndex)
				if err != nil {
					return
				}
			}

			downloader.resolveRequest(request, timeout)

			downloader.sendRequest(peer, request)
		}
	}

	// sending fresh requests

	requests := downloader.selectedPiece.getNextRequest(maxPendingRequest - len(downloader.pendingRequest))

	log.Debugf("retrieve %v reqs", len(requests))

	for i, request := range requests {

		log.Debugf("%v- request id: %v", i,request.Id)

		if peer == nil || !peer.isAvailable(request.Id) {

			//if the current peer is not available we try to select another peer
			peer, err = downloader.peerManager.GetAvailablePeer(request.Id,downloader.selectedPiece.PieceIndex)

			if err != nil {
				// if we still can't find a peer, we still add the request to to the pending requests map,  hasNoProvider() will return true, the next time at the next iteration fo the loop
				downloader.pendingRequest[request.Id] = request
				continue
			}
		}
		downloader.sendRequest(peer, request)

	}

}

/*
	Sends  pieceRequests to a Peer
*/
func (downloader *Downloader) sendRequest(peer *Peer, pieceRequest ...*BlockRequest) {

	for i, request := range pieceRequest {
		log.Debugf("sending piece req to %v, requestID %v",request.BeginIndex,request.Id)
		_, err := peer.SendMsg(request.Marshal())

		if err != nil {
			log.Fatal(err)
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
		if downloader.selectedPiece == nil {
			index = 0
		} else {
			index = downloader.selectedPiece.PieceIndex + 1
		}



		downloader.selectedPiece = downloader.Pieces[index]
		return downloader.selectedPiece, nil

	}
	selectedPiece, err := downloader.queue.RemoveAt(index, downloader.mutex)

	downloader.selectedPiece = selectedPiece
	if err != nil {
		return nil, err
	}
	return selectedPiece, err

}

func (downloader *Downloader) PutPiece(msg PieceMsg) {
	blocKReqId := getRequestId(msg.PieceIndex, msg.BeginIndex)
	msg.GetPeer().resolvedRequest(blocKReqId)


	if !downloader.selectedPiece.hasSubPiece(msg.BeginIndex) {
		log.Infof("adding block.. start index %v, payload len %v; sent by %v",msg.BeginIndex,len(msg.Payload), msg.GetPeer().GetId())

		isComplete := downloader.writer.writeToBuffer(msg.Payload, msg.BeginIndex)

		downloader.resolveRequest(downloader.pendingRequest[blocKReqId], fulfilled)

		if isComplete {

			log.Infof("piece %v is complete",downloader.selectedPiece.PieceIndex)

			err := downloader.writer.writePiece()
			if err != nil {
				os.Exit(233993)
				return
			}

			downloader.torrentManager.SyncOperation <- func() {

				log.Debugf("selecting new piece....")
				piece, err := downloader.selectPiece()
				if err != nil {
					return
				}

				downloader.writer.setNewPiece(piece.pieceLength, piece.PieceIndex)
				downloader.selectedPiece = piece
				log.Debugf("selected new piece: id %v", downloader.selectedPiece.PieceIndex)

			}

		}
	} else {

		log.Debugf("already have piece: index %v, begin index %v", downloader.selectedPiece.PieceIndex,msg.BeginIndex)
	}

}

func (downloader *Downloader) resolveRequest(blockRequest *BlockRequest, action int) {
	atomic.StoreInt64(&blockRequest.state, int64(action))

	if action == fulfilled {
		log.Debugf("request %v, has been fulffiled", blockRequest.Id)
		downloader.selectedPiece.UpdateBitfield(blockRequest.BeginIndex)
		blockRequest.Providers = blockRequest.Providers[len(blockRequest.Providers):]

	}else if action == timeout{
		log.Debugf("request %v, has been timedout", blockRequest.Id)

	}

	delete(downloader.pendingRequest, blockRequest.Id)

}