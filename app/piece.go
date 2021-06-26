package app

import (
	"DawnTorrent/interfaces"
	"DawnTorrent/utils"
	log "github.com/sirupsen/logrus"
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
	buffer          []byte
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
	endOfPiece 			bool
	nPendingRequest	int
	writer PieceWriter
	blockChan chan PieceMsg
}

func (piece Piece) getBlockRequestInfo(index int) (int, int, bool) {

	startIndex := BlockLen * (index)

	currentTotalLength := BlockLen + startIndex

	if currentTotalLength > piece.pieceLength {
		currentTotalLength = startIndex + (piece.pieceLength%startIndex)
	}

	return startIndex, BlockLen, currentTotalLength==piece.pieceLength
}

func (piece *Piece) getNextRequest(nRequest int) []*BlockRequest {
	var requests []*BlockRequest

	if piece.endOfPiece {
		return requests
	}

	for i := piece.requestIndex; i < piece.requestIndex + nRequest; i++ {

		startIndex, currentSubPieceLength, endOfPiece := piece.getBlockRequestInfo(i)

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

		if endOfPiece {
			piece.requestIndex += len(requests)
			piece.endOfPiece = true
			return requests
		}

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

func (piece *Piece) download(peer *Peer) error {
	piece.buffer = make([]byte,piece.pieceLength)

	_, err := peer.connection.Write(InterestedMsg{
		header{
			ID: InterestedMsgId,
		},
	}.Marshal())
	if err != nil {
		return err
	}
	for  piece.downloaded < piece.pieceLength {
		requests := piece.getNextRequest(maxPendingRequest-piece.nPendingRequest)

		for _, request := range requests {

			//log.Info("sending request ...")
			_, err2 := peer.GetConnection().Write(request.Marshal())
			if err2 != nil {
				return err2
			}
				 piece.nPendingRequest++

			

		}
	}

	log.Infof("completed piece %v", piece.PieceIndex)

	return nil
}

func (piece *Piece) putPiece(msg PieceMsg){
	log.Debug("putting piece....")
	if !piece.hasSubPiece(msg.BeginIndex) && piece.downloaded < piece.pieceLength{
		piece.nPendingRequest--
		piece.writeToBuffer(msg.Payload,msg.BeginIndex)

	}
}



func (piece *Piece) writeToBuffer(data []byte, startIndex int) bool {
	if piece.downloaded < piece.pieceLength {
		subPieceLen := len(data)
		copy(piece.buffer[startIndex:startIndex+subPieceLen], data)
		piece.downloaded += subPieceLen
	}
	return piece.downloaded == piece.pieceLength
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