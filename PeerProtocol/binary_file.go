package PeerProtocol

import (
	"crypto/sha1"
	"encoding/hex"
	"math"
	"sort"
	"strconv"
	"sync"
	"torrent/parser"
)

type torrentFile struct {
	infoHash            string
	fileName            string
	fileLen             int
	currentPiece        *Piece
	Pieces              []*Piece
	PiecesMutex         *sync.RWMutex
	SortedAvailability  []int
	nPiece              int
	nSubPiece           int
	SubPieceLen         int
	PieceLen            int
	completedPieceIndex map[int]*Piece
	neededPieceMap      map[int]*Piece
	behavior            string
	torrentMetaInfo     *parser.Dict
}

func NewFile(torrent *Torrent, torrentPath string) *torrentFile {
	torrentFile := new(torrentFile)
	torrentFile.PiecesMutex = new(sync.RWMutex)
	torrentFile.torrentMetaInfo = parser.Unmarshall(torrentPath)
	torrentFile.infoHash = GetInfoHash(torrentFile.torrentMetaInfo)
	torrentFile.fileLen, _ = strconv.Atoi(torrentFile.torrentMetaInfo.MapDict["info"].MapString["length"])
	torrentFile.PieceLen, _ = strconv.Atoi(torrentFile.torrentMetaInfo.MapDict["info"].MapString["piece length"])
	torrentFile.SubPieceLen = int(math.Min(float64(torrentFile.PieceLen), 16000))

	torrentFile.nPiece = int(math.Ceil(float64(torrentFile.fileLen) / float64(torrentFile.PieceLen)))
	torrentFile.nSubPiece = int(math.Ceil(float64(torrentFile.PieceLen) / float64(SubPieceLen)))
	torrentFile.Pieces = make([]*Piece, torrentFile.nPiece)
	torrentFile.neededPieceMap = make(map[int]*Piece)
	//TODO this will probably be removed
	torrentFile.completedPieceIndex = make(map[int]*Piece)
	torrentFile.SortedAvailability = make([]int, torrentFile.nPiece)
	/// it's weird not sure it is the right way

	for i, _ := range torrentFile.Pieces {
		pieceLen := torrentFile.PieceLen
		if i == torrentFile.nPiece-1 {
			if float64(torrentFile.fileLen)/float64(torrentFile.fileLen) != 0 {
				pieceLen = torrentFile.PieceLen % torrentFile.PieceLen
			}
		}
		torrentFile.Pieces[i] = torrentFile.NewPiece(pieceLen, i)
		torrentFile.neededPieceMap[i] = torrentFile.Pieces[i]
		torrentFile.SortedAvailability[i] = i
	}

	torrentFile.behavior = "random"
	return torrentFile
}

func GetInfoHash(dict *parser.Dict) string {

	// InnerStartingPosition leaves out the key

	startingPosition := dict.MapDict["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.MapDict["info"].KeyInfo.EndingPosition

	torrentFileString := parser.ToBencode(dict)

	torrentFileByte := []byte(torrentFileString)

	infoBytes := torrentFileByte[startingPosition:endingPosition]

	b := sha1.Sum(infoBytes)
	bSlice := b[:]

	println(hex.EncodeToString(bSlice))

	return string(bSlice)

}
func (torrentFile *torrentFile) NewPiece(PieceTotalLen int, pieceIndex int) *Piece {
	newPiece := new(Piece)

	newPiece.PieceTotalLen = PieceTotalLen
	newPiece.owners = make(map[string]*Peer)
	newPiece.Availability = 0
	newPiece.PieceIndex = pieceIndex
	newPiece.SubPieces = make([][]byte, torrentFile.nSubPiece)
	newPiece.Status = "empty"
	for i, _ := range newPiece.SubPieces {
		newPiece.SubPieces[i] = make([]byte, 0)
	}

	return newPiece

}
func (torrentFile *torrentFile) Len() int {
	return len(torrentFile.SortedAvailability)
}
func (torrentFile *torrentFile) Less(i, j int) bool {

	return false
}
func (torrentFile *torrentFile) Swap(i, j int) {
	temp := torrentFile.SortedAvailability[i]
	torrentFile.SortedAvailability[i] = torrentFile.SortedAvailability[j]
	torrentFile.SortedAvailability[j] = temp

}
func (torrentFile *torrentFile) sortPieces() {
	sort.Sort(torrentFile)
}

type Piece struct {
	PieceTotalLen int
	CurrentLen    int
	SubPieceLen   int
	Status        string
	SubPieces     [][]byte
	PieceIndex    int
	Availability  int
	owners        map[string]*Peer
}
