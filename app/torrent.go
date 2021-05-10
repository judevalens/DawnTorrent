package app

import (
	"DawnTorrent/parser"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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
	pieces []*Piece
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
	torrentFile := new(Torrent)

	infoHashByte, hexInfoHash := GetInfoHash(torrentMap)

	torrentFile.infoHashByte = infoHashByte
	torrentFile.InfoHashHex = hexInfoHash
	torrentFile.InfoHash = string(torrentFile.infoHashByte[:])
	torrentFile.AnnouncerUrl = torrentMap.Strings["announce"]
	torrentFile.AnnounceList = make([]string, len(torrentMap.BLists["announce-list"].BLists))

	for i, announcerUrl := range torrentMap.BLists["announce-list"].BLists {
		fmt.Printf("url : %v\n", announcerUrl.Strings)
		if len(announcerUrl.Strings) > 0 {
			torrentFile.AnnounceList[i] = announcerUrl.Strings[0]
		}
	}

	torrentFile.CreationDate = torrentMap.Strings["creation date"]
	torrentFile.Encoding = torrentMap.Strings["encoding"]
	torrentFile.FileLength, _ = strconv.Atoi(torrentMap.BMaps["info"].Strings["length"])
	torrentFile.piecesHash = torrentMap.BMaps["info"].Strings["pieces"]
	torrentFile.pieceLength, _ = strconv.Atoi(torrentMap.BMaps["info"].Strings["piece length"])

	_, isMultipleFiles := torrentMap.BMaps["info"].BLists["files"]
	var fileProperties []fileMetadata
	torrentFile.Name = torrentMap.BMaps["info"].Strings["name"]
	totalLength := 0

	if isMultipleFiles {
		torrentFile.FileMode = MultipleFile
		for fileIndex, v := range torrentMap.BMaps["info"].BLists["files"].BMaps {
			filePath := v.BLists["path"].Strings[0]
			fileLength, _ := strconv.Atoi(v.Strings["length"])
			totalLength += fileLength
			newFile := createFileProperties(fileProperties, filePath, fileLength, fileIndex)
			fileProperties = append(fileProperties, newFile)
		}
	} else {
		torrentFile.FileMode = SingleFile
		filePath := torrentMap.BMaps["info"].Strings["name"]
		fileLength, _ := strconv.Atoi(torrentMap.BMaps["info"].Strings["length"])
		totalLength += fileLength
		fileProperties = []fileMetadata{createFileProperties(fileProperties, filePath, fileLength, 0)}
	}
	torrentFile.buildPieces()

	return torrentFile, nil
}

func (torrent *Torrent) buildPieces()  {
	for i,_ := range torrent.pieces {

		pieceLen := torrent.pieceLength

		startIndex := i*torrent.pieceLength
		currentTotalLength := startIndex+pieceLen

		if currentTotalLength > torrent.FileLength{
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
	pieceLength     int
	CurrentLen      int
	SubPieceLen     int
	PieceIndex      int
	State           int
	Pieces          []byte
	QueueIndex      int
	Availability    int
	subPieceMask    []byte
	pieceStartIndex int
	pieceEndIndex   int
	position            []int
	pendingRequestMutex *sync.RWMutex

	nSubPiece         int
	AvailabilityIndex int
}

func (piece Piece) getSubPieceLength(index int) (int,int,bool) {

	startIndex := piece.CurrentLen+(subPieceLen*(index))
	currentTotalLength := piece.CurrentLen+(subPieceLen*(index + 1))

	if currentTotalLength > piece.pieceLength{
		return  startIndex,currentTotalLength % piece.pieceLength,true
	}

	return  startIndex,subPieceLen,false
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
			torrentMsg : header{
				RequestMsgId,
				msgLength,
				nil,
			},
			PieceIndex: piece.PieceIndex,
			BeginIndex: startIndex,
			BlockLength: currentSubPieceLength,
		}

		req := &pieceRequest{
			fullFilled: false,
			RequestMsg:msg,
			providers: make([]*Peer,0),
		}

		requests = append(requests,req)

	}

	return requests
}

//	Creates a Piece object and initialize subPieceRequest for this piece

func NewPiece(downloader *Torrent, PieceIndex, pieceLength int, status int) *Piece {
	newPiece := new(Piece)

	newPiece.pieceLength = pieceLength

	newPiece.Availability = 0
	newPiece.PieceIndex = PieceIndex
	newPiece.pieceStartIndex = PieceIndex * downloader.pieceLength
	newPiece.pieceEndIndex = newPiece.pieceStartIndex + newPiece.pieceLength
	newPiece.nSubPiece = int(math.Ceil(float64(pieceLength) / float64(downloader.subPieceLength)))
	//newPiece.Pieces = make([]byte, newPiece.pieceLength)
	newPiece.State = status
	newPiece.subPieceMask = make([]byte, int(math.Ceil(float64(newPiece.nSubPiece)/float64(8))))
	newPiece.pendingRequestMutex = new(sync.RWMutex)

	for i := 0; i < newPiece.nSubPiece; i++ {
		newPiece.SubPieceLen = downloader.subPieceLength
		if i == newPiece.nSubPiece-1 {
			if newPiece.pieceLength%downloader.subPieceLength != 0 {
				newPiece.SubPieceLen = newPiece.pieceLength % downloader.subPieceLength
				println("newPiece.SubPieceLen")
				println(newPiece.SubPieceLen)
				fmt.Printf("pieceLen %v\n", newPiece.pieceLength)

				if newPiece.pieceLength != 1048576 {

				}
			}
		}
	}

	return newPiece

}

