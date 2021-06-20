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
	owners            map[string]interfaces.PeerI
	mask              int
	nSubPiece         int
	AvailabilityIndex int
	requestIndex 		int
}

func (piece Piece) getBlockRequestInfo(index int) (int, int, bool) {

	startIndex := BlockLen * (index)

	currentTotalLength := BlockLen * (index + 1)

	if currentTotalLength > piece.pieceLength {
		return startIndex, currentTotalLength - piece.pieceLength, true
	}

	return startIndex, BlockLen, false
}

func (piece *Piece) getNextRequest(nRequest int) []*BlockRequest {
	var requests []*BlockRequest

	for i := piece.requestIndex; i < piece.requestIndex + nRequest; i++ {

		startIndex, currentSubPieceLength, endOfPiece := piece.getBlockRequestInfo(i)

		if endOfPiece {
			piece.requestIndex += len(requests)
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
			state:      unfulfilled,
			RequestMsg: msg,
			Providers:  make([]*Peer, 0),
		}

		requests = append(requests, req)

	}

	piece.requestIndex += len(requests)

	return requests
}

//	Increments a piece availability is a peer a possesses it, decrements it if the peer is choking or has disconnected
func (piece Piece) updateAvailability(action int, peer interfaces.PeerI) {

	if action == incrementAvailability {
		// avoids increasing a piece availability multiple times for the same peer
		_, found := piece.owners[peer.GetId()]

		if peer.HasPiece(piece.PieceIndex) && !found {
			atomic.AddInt64(&piece.Availability, 1)
			piece.owners[peer.GetId()] = peer

			// will be used to determine if a peer is interesting
			if piece.State == notStarted {
				peer.SetInterestPoint(peer.GetInterestPoint()+1)
			}

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
	newPiece.owners = make(map[string]interfaces.PeerI)
	newPiece.mutex = new(sync.Mutex)

	maskSize  := int(math.Ceil(float64(newPiece.pieceLength) / BlockLen))

	newPiece.subPieceMask= make([]byte,maskSize)
	return newPiece

}

func (piece *Piece) buildBitfield() {
	piece.subPieceMask = make([]byte, int(math.Ceil(float64(piece.nSubPiece)/float64(8))))
}

func (piece *Piece) hasSubPiece(startIndex int) bool {
	subPieceIndex := startIndex/BlockLen

	byteIndex := subPieceIndex / 8

	bitIndex := 7 - (subPieceIndex%8)

	return utils.IsBitOn(piece.subPieceMask[byteIndex], bitIndex)
}

func (piece *Piece) UpdateBitfield(startIndex int) {
	subPieceIndex := startIndex/BlockLen

	byteIndex := subPieceIndex / 8

	bitIndex := 7 - (subPieceIndex%8)
	piece.subPieceMask[byteIndex] = utils.BitMask(piece.subPieceMask[byteIndex], 1, bitIndex)
}

