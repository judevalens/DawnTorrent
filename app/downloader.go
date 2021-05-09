package app

import (
	"container/heap"
	"time"
)

const (
	notStarted = iota
	inProgress = iota
	completed = iota
)

const (
	random = iota
	rarestFirst = iota
	endGame = iota
)

type downloaderState interface {
	
}

type pendingPiece []*Piece

func (p pendingPiece) Len() int {
	return len(p)
}

func (p pendingPiece) Less(i, j int) bool {
	return p[i].Availability < p[j].Availability
}

func (p pendingPiece) Swap(i, j int) {
	p[i],p[j] = p[j], p[i]
}

func (p *pendingPiece) Push(x interface{}) {
	piece :=  x.(*Piece)
	piece.QueueIndex = len(*p)
	*p = append(*p, piece)
}

func (p *pendingPiece) Pop() interface{} {
	old := *p
	l := len(*p)
	item := old[l-1]
	old[l-1] = nil
	item.QueueIndex = -1
	*p = old[0:l-1]
	return item
}

type torrentDownloader struct {
	*pendingPiece
	torrent Torrent
	torrentManager torrentManagerStateI
	mode int
}

func newTorrentDownloader () torrentDownloader{
	return torrentDownloader{}
}

func (downloader *torrentDownloader) download()  {
}


func (downloader *torrentDownloader) updatePiecePriority(i int){
		heap.Fix(downloader.pendingPiece,i)
}

func (downloader *torrentDownloader) selectPiece(){

	if downloader.mode == random{

	}else if downloader.mode == rarestFirst{

	}

}

func (downloader *torrentDownloader) write(msg PieceMsg){

}


type pieceRequestObserver interface {
	notify()
}

type pieceRequest struct {
	fullFilled bool
	RequestMsg
	timeout		time.Duration
	providers []*Peer
}