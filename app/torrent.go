package app

import (
	"DawnTorrent/parser"
	"DawnTorrent/protocol"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
)

const (
	SingleFile   = iota
	MultipleFile = iota
)

type Torrent struct {
	AnnouncerUrl   string
	AnnounceList   []string
	Comment        string
	CreatedBy      string
	CreationDate   string
	Encoding       string
	piecesHash     string
	FileMode       int
	FilesMetadata  []fileMetadata
	nPiece         int
	FileLength     int
	subPieceLength int
	nSubPiece      int
	infoHashByte   [20]byte
	InfoHashHex    string
	InfoHash       string
	pieceLength    int
	Name           string
	pieces         []*Piece
	bitfield 	[]byte
}

type fileMetadata struct {
	Path       string
	Length     int
	FileIndex  int
	StartIndex int
	EndIndex   int
}

func createNewTorrent(torrentPath string) (*Torrent, error) {
	torrentMap, err := parser.Unmarshall(torrentPath)
	if err != nil {
		return nil, err
	}
	torrent := new(Torrent)

	infoHashByte, hexInfoHash := GetInfoHash(torrentMap)

	torrent.infoHashByte = infoHashByte
	torrent.InfoHashHex = hexInfoHash
	torrent.InfoHash = string(torrent.infoHashByte[:])
	torrent.AnnouncerUrl = torrentMap.Strings["announce"]
	torrent.AnnounceList = make([]string, len(torrentMap.BLists["announce-list"].BLists))

	for i, announcerUrl := range torrentMap.BLists["announce-list"].BLists {
		fmt.Printf("url : %v\n", announcerUrl.Strings)
		if len(announcerUrl.Strings) > 0 {
			torrent.AnnounceList[i] = announcerUrl.Strings[0]
		}
	}

	torrent.CreationDate = torrentMap.Strings["creation date"]
	torrent.Encoding = torrentMap.Strings["encoding"]
	torrent.FileLength, _ = strconv.Atoi(torrentMap.BMaps["info"].Strings["length"])
	torrent.piecesHash = torrentMap.BMaps["info"].Strings["pieces"]
	torrent.pieceLength, _ = strconv.Atoi(torrentMap.BMaps["info"].Strings["piece length"])
	torrent.Name = torrentMap.BMaps["info"].Strings["name"]
	torrent.buildFileMetadata(torrentMap)
	torrent.buildPieces()
	// initializing bitfield
	bitfieldLen := int(math.Ceil(float64(torrent.nPiece) / 8))

	log.Printf("bitfield len %v",bitfieldLen)
	torrent.bitfield = make([]byte,bitfieldLen)


	return torrent, nil
}

func(torrent *Torrent) buildFileMetadata(torrentMap *parser.BMap)  {
	_, isMultipleFiles := torrentMap.BMaps["info"].BLists["files"]
	var fileProperties []fileMetadata

	if isMultipleFiles {
		torrent.FileMode = MultipleFile
		for fileIndex, v := range torrentMap.BMaps["info"].BLists["files"].BMaps {
			filePath := v.BLists["path"].Strings[0]
			fileLength, _ := strconv.Atoi(v.Strings["length"])
			torrent.FileLength += fileLength
			newFile := createFileProperties(fileProperties, filePath, fileLength, fileIndex)
			fileProperties = append(fileProperties, newFile)
		}
	} else {
		torrent.FileMode = SingleFile
		filePath := torrentMap.BMaps["info"].Strings["name"]
		fileLength, _ := strconv.Atoi(torrentMap.BMaps["info"].Strings["length"])
		torrent.FileLength += fileLength
		fileProperties = []fileMetadata{createFileProperties(fileProperties, filePath, fileLength, 0)}
	}

	torrent.FilesMetadata = fileProperties
}

func (torrent *Torrent) buildPieces() {
	torrent.nPiece = int(math.Ceil(float64(torrent.FileLength) / float64(torrent.pieceLength)))

	log.Printf("file length %v, pieceLength %v, n pieces %v", torrent.FileLength,torrent.pieceLength, torrent.nPiece)

	torrent.pieces = make([]*Piece,torrent.nPiece)
	for i, _ := range torrent.pieces {
		pieceLen := torrent.pieceLength

		startIndex := i * torrent.pieceLength
		currentTotalLength := startIndex + pieceLen

		if currentTotalLength > torrent.FileLength {
			pieceLen = currentTotalLength % torrent.FileLength
		}

		torrent.pieces[i] = NewPiece(torrent, i, pieceLen, notStarted)
	}

}


// store the path , len , start and end index of file
func createFileProperties(fileProperties []fileMetadata, filePath string, fileLength int, fileIndex int) fileMetadata {
	torrent := fileMetadata{}

	torrent.Path = filePath
	torrent.Length = fileLength

	if fileIndex == 0 {
		torrent.StartIndex = 0
	} else {
		torrent.StartIndex = fileProperties[fileIndex-1].EndIndex
	}

	// end index of file is not inclusive
	torrent.EndIndex = torrent.StartIndex + torrent.Length

	return torrent
}

// GetInfoHash calculates the info hash based on pieces provided in the .torrent file
func GetInfoHash(dict *parser.BMap) ([20]byte, string) {

	// InnerStartingPosition leaves out the key

	startingPosition := dict.BMaps["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.BMaps["info"].KeyInfo.EndingPosition

	torrentFileString := parser.ToBencode(dict)

	torrentFileByte := []byte(torrentFileString)

	infoBytes := torrentFileByte[startingPosition:endingPosition]

	infoHash := sha1.Sum(infoBytes)
	infoHashSlice := infoHash[:]
	println(hex.EncodeToString(infoHashSlice))

	return infoHash, hex.EncodeToString(infoHashSlice)

}

type Piece struct {
	pieceLength         int
	CurrentLen          int
	SubPieceLen         int
	PieceIndex          int
	State               int
	Pieces              []byte
	QueueIndex          int
	Availability        int
	subPieceMask        []byte
	pieceStartIndex     int
	pieceEndIndex       int
	position            []int
	pendingRequestMutex *sync.RWMutex
	owners              map[string]bool
	mask                int
	nSubPiece           int
	AvailabilityIndex   int
}

func (piece Piece) getSubPieceLength(index int) (int, int, bool) {

	startIndex := piece.CurrentLen + (subPieceLen * (index))
	currentTotalLength := piece.CurrentLen + (subPieceLen * (index + 1))

	if currentTotalLength > piece.pieceLength {
		return startIndex, currentTotalLength % piece.pieceLength, true
	}

	return startIndex, subPieceLen, false
}

func (piece Piece) getNextRequest(nRequest int) []*pieceRequest {
	var requests []*pieceRequest

	for i := 0; i < nRequest; i++ {
		startIndex, currentSubPieceLength, endOfPiece := piece.getSubPieceLength(i)

		if endOfPiece {
			return requests
		}

		msgLength := requestMsgLen + currentSubPieceLength

		msg := RequestMsg{
			torrentMsg: header{
				RequestMsgId,
				msgLength,
				nil,
			},
			PieceIndex:  piece.PieceIndex,
			BeginIndex:  startIndex,
			BlockLength: currentSubPieceLength,
		}

		req := &pieceRequest{
			fullFilled: false,
			RequestMsg: msg,
			providers:  make([]*Peer, 0),
		}

		requests = append(requests, req)

	}

	return requests
}


//	Increments a piece availability is a peer a possesses it, decrements it if the peer is choking or has disconnected
func (piece Piece)  updateAvailability(action int,peer protocol.PeerI){
	peer.GetMutex().Lock()
	if action == 1 {
		if peer.HasPiece(piece.PieceIndex){
			piece.Availability++
			piece.owners[peer.GetId()] = true

			log.Printf("peer has piece # %v\n",piece.PieceIndex)
		}else{
			log.Printf("peer doesnt have piece # %v\n",piece.PieceIndex)
		}
	}else{
		if peer.HasPiece(piece.PieceIndex){
			piece.Availability--
			delete(piece.owners,peer.GetId())

		}
	}
	peer.GetMutex().Unlock()

}

//	Creates a Piece object and initialize subPieceRequest for this piece

func NewPiece(downloader *Torrent, PieceIndex, pieceLength int, status int) *Piece {
	newPiece := new(Piece)

	newPiece.pieceLength = pieceLength

	newPiece.Availability = 0
	newPiece.PieceIndex = PieceIndex
	newPiece.pieceStartIndex = PieceIndex * downloader.pieceLength
	newPiece.pieceEndIndex = newPiece.pieceStartIndex + newPiece.pieceLength
	newPiece.State = status
	newPiece.owners = make(map[string]bool)

	return newPiece

}
