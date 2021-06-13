package app

import (
	"DawnTorrent/interfaces"
	"DawnTorrent/utils"
	"math"
	"sync"
	"sync/atomic"
)

type Piece struct {
	pieceLength     int
	downloaded      int
	SubPieceLen     int
	PieceIndex      int
	State           int
	Pieces          []byte
	QueueIndex      int
	Availability    int64
	subPieceMask    []byte
	pieceStartIndex int
	pieceEndIndex   int
	position        []int
	mutex             *sync.Mutex
	owners            map[string]bool
	mask              int
	nSubPiece         int
	AvailabilityIndex int
}

func (piece Piece) getSubPieceLength(index int) (int, int, bool) {

	startIndex := BlockLen * (index)

	currentTotalLength := BlockLen * (index + 1)

	if currentTotalLength > piece.pieceLength {
		return startIndex, currentTotalLength % piece.pieceLength, true
	}

	return startIndex, BlockLen, false
}

func (piece Piece) getNextRequest(nRequest int) []*BlockRequest {
	var requests []*BlockRequest

	for i := 0; i < nRequest; i++ {
		startIndex, currentSubPieceLength, endOfPiece := piece.getSubPieceLength(i)

		if endOfPiece {
			return requests
		}

		msgLength := requestMsgLen + currentSubPieceLength

		msg := RequestMsg{
			TorrentMsg: header{
				RequestMsgId,
				msgLength,
				nil,
			},
			PieceIndex:  piece.PieceIndex,
			BeginIndex:  startIndex,
			BlockLength: currentSubPieceLength,
		}

		req := &BlockRequest{
			Id:         getRequestId(piece.PieceIndex, startIndex),
			FullFilled: false,
			RequestMsg: msg,
			Providers:  make([]*Peer, 0),
		}

		requests = append(requests, req)

	}

	return requests
}

//	Increments a piece availability is a peer a possesses it, decrements it if the peer is choking or has disconnected
func (piece Piece) updateAvailability(action int, peer interfaces.PeerI) {

	if action == 1 {

		if peer.HasPiece(piece.PieceIndex) {
			atomic.AddInt64(&piece.Availability, 1)
			piece.owners[peer.GetId()] = true

		} else {
		}
	} else {
		if peer.HasPiece(piece.PieceIndex) {
			atomic.AddInt64(&piece.Availability, -1)
			delete(piece.owners, peer.GetId())

		}
	}

}

//	Creates a Piece object and initialize subPieceRequest for this piece

func NewPiece(PieceIndex, standardPieceLength, pieceLength int, status int) *Piece {
	newPiece := new(Piece)

	newPiece.pieceLength = pieceLength

	newPiece.Availability = 0
	newPiece.PieceIndex = PieceIndex
	newPiece.pieceStartIndex = PieceIndex * standardPieceLength
	newPiece.pieceEndIndex = newPiece.pieceStartIndex + newPiece.pieceLength
	newPiece.State = status
	newPiece.owners = make(map[string]bool)
	newPiece.mutex = new(sync.Mutex)

	return newPiece

}

func (piece *Piece) buildBitfield() {
	piece.subPieceMask = make([]byte, int(math.Ceil(float64(piece.nSubPiece)/float64(8))))
}

func (piece *Piece) hasSubPiece(startIndex int) bool {
	byteIndex := (startIndex * BlockLen) / 8

	bitIndex := 7 - byteIndex%8

	return utils.IsBitOn(piece.subPieceMask[byteIndex], bitIndex)
}

func (piece *Piece) UpdateBitfield(startIndex int) {
	byteIndex := startIndex / 8
	bitIndex := 7 - byteIndex%8
	piece.subPieceMask[byteIndex] = utils.BitMask(piece.subPieceMask[byteIndex], 1, bitIndex)
}

