package app

import (
	"container/heap"
	"context"
	"errors"
	"log"
	"math/rand"
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
	queueLength = 5
	subPieceLen = 1024
	maxReq = 10
	requestTimeOut = time.Second*2
)



type downloaderState interface {
}

type downloadQueue []*Piece

func (p *downloadQueue) RemoveAt(i int) (*Piece, error) {
	if i > len(*p) {
		return nil, errors.New("i > len(x)")
	}
	old := *p
	item := old[i]
	*p = append(old[:i], old[:i+1]...)
	heap.Fix(p, i)
	return item, nil
}

func (p downloadQueue) Len() int {
	return len(p)
}

func (p downloadQueue) Less(i, j int) bool {
	return p[i].Availability < p[j].Availability
}

func (p downloadQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p *downloadQueue) Push(x interface{}) {
	piece := x.(*Piece)
	piece.QueueIndex = len(*p)
	*p = append(*p, piece)
}

func (p *downloadQueue) Pop() interface{} {
	l := len(*p)
	if l == 0 {
		return nil
	}
	old := *p
	item := old[l-1]
	old[l-1] = nil
	item.QueueIndex = -1
	*p = old[0 : l-1]
	return item
}

type torrentDownloader struct {
	*downloadQueue
	torrent        Torrent
	torrentManager torrentManagerStateI
	peerManager    *peerManager
	mode           int
	selectedPiece  *Piece
	queue 			[]*pieceRequest
	ticker 			chan time.Time

}

func newTorrentDownloader() torrentDownloader {
	return torrentDownloader{}
}

func (downloader *torrentDownloader) start(ctx context.Context){
	for {
		select {
		case <- ctx.Done():
			return
		case <- downloader.ticker:
			downloader.download(ctx)
			// this basically calls the download method after a certain amount of time
			time.AfterFunc(time.Millisecond*150, func() {
				downloader.ticker<- time.Now()
			})

		}
	}
}


func (downloader *torrentDownloader) download(ctx context.Context) {

	var err error
	currentPiece, err := downloader.selectPiece()

	if err != nil {
		log.Fatal(err)
	}




	// resend requests that had timed out
	availablePeer , err := 	downloader.peerManager.getAvailablePeer()

	for _, request := range downloader.queue {

		if time.Now().Sub(request.timeStamp) > requestTimeOut{

			peer, err := downloader.peerManager.getAvailablePeer()

			downloader.sendRequest(peer,request)

			if err != nil {
				return
			}

		}
		request.get()
	}


	// sending fresh request
	availablePeer , err = 	downloader.peerManager.getAvailablePeer()

	for err != nil{
		requests := currentPiece.getNextRequest(10)

		downloader.sendRequest(availablePeer,requests...)
		availablePeer , err = 	downloader.peerManager.getAvailablePeer()
	}

}




func  (downloader *torrentDownloader) sendRequest(peer *Peer,pieceRequest ...*pieceRequest){

	for i, request := range pieceRequest {
		_, err := peer.sendMsg(request.marshal())
		if err != nil {
			return
		}
		// each request must be completed in at least requestTimeOut
		request.timeStamp = time.Now().Add(requestTimeOut* time.Duration(int64(i)*time.Nanosecond.Nanoseconds()))
		request.providers = append(request.providers,peer)
	}

}

func (downloader *torrentDownloader) updatePiecePriority(i int) {
	heap.Fix(downloader.downloadQueue, i)
}

func (downloader *torrentDownloader) selectPiece() (*Piece, error) {

	if downloader.selectedPiece.State == completed {
		return downloader.selectedPiece, nil
	}

	var index int
	var selectedPiece *Piece
	if downloader.mode == random {
		randGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
		index = randGenerator.Intn(downloader.downloadQueue.Len())

	} else if downloader.mode == rarestFirst {
		selectedPiece = downloader.downloadQueue.Pop().(*Piece)
		if selectedPiece == nil {
			return nil, errors.New("queue is empty")
		}
		return selectedPiece, nil
	}
	selectedPiece, err := downloader.RemoveAt(index)
	if err != nil {
		return nil, err
	}
	return selectedPiece, err

}

func (downloader *torrentDownloader) write(msg PieceMsg) {

}

type pieceRequestObserver interface {
	notify()
}

type pieceRequest struct {
	fullFilled bool
	RequestMsg
	timeStamp		time.Time
	providers []*Peer
}

func (request RequestMsg) get()  {

}
