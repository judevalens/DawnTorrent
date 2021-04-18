package protocol

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

const(
	NotStarted = iota
	InProgress = iota
	Completed = iota
)

type Torrent struct {
	Announce           string
	AnnounceList       []string
	Comment            string
	CreatedBy          string
	CreationDate       string
	Encoding           string
	piecesHash         string
	FileMode           int
	FilesMetadata      []fileMetadata
	nPiece             int
	FileLength         int
	subPieceLength     int
	nSubPiece          int
	infoHashByte       [20]byte
	InfoHashHex        string
	pieceLength        int
	Name               string
	PieceHolder        []*byte
	PieceSelectionMode int
}

type fileMetadata struct {
	Path       string
	Length     int
	FileIndex  int
	StartIndex int
	EndIndex   int
}

func loadTorrentFile(filePath string) *parser.BMap {
	return parser.Unmarshall(filePath)
}

func createNewTorrent(torrentMap *parser.BMap) *Torrent {
	torrentFile := new(Torrent)

	infoHashByte, hexInfoHash := GetInfoHash(torrentMap)

	torrentFile.infoHashByte = infoHashByte
	torrentFile.InfoHashHex = hexInfoHash
	torrentFile.Announce = torrentMap.Strings["announce"]

	torrentFile.AnnounceList = make([]string, 0)
	torrentFile.AnnounceList = torrentMap.BLists["announce-list"].Strings
	torrentFile.CreationDate = torrentMap.Strings["creation date"]
	torrentFile.Encoding = torrentMap.Strings["encoding"]
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
	return torrentFile
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

// calculates the info hash based on pieces provided in the .torrent file
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
	Len                 int
	CurrentLen          int
	SubPieceLen         int
	State               int
	Pieces              []byte
	PieceIndex          int
	Availability        int
	subPieceMask        []byte
	pieceStartIndex     int
	pieceEndIndex       int
	position            []pos
	pendingRequestMutex *sync.RWMutex

	nSubPiece           int
	AvailabilityIndex   int
}

//	Creates a Piece object and initialize subPieceRequest for this piece
func NewPiece(downloader *Torrent, PieceIndex, pieceLength int, status int) *Piece {
	newPiece := new(Piece)

	newPiece.Len = pieceLength
	newPiece.Availability = 0
	newPiece.PieceIndex = PieceIndex
	newPiece.pieceStartIndex = PieceIndex * downloader.pieceLength
	newPiece.pieceEndIndex = newPiece.pieceStartIndex + newPiece.Len
	newPiece.nSubPiece = int(math.Ceil(float64(pieceLength) / float64(downloader.subPieceLength)))
	//newPiece.Pieces = make([]byte, newPiece.Len)
	newPiece.State = status
	newPiece.subPieceMask = make([]byte, int(math.Ceil(float64(newPiece.nSubPiece)/float64(8))))
	newPiece.pendingRequestMutex = new(sync.RWMutex)

	for i := 0; i < newPiece.nSubPiece; i++ {
		newPiece.SubPieceLen = downloader.subPieceLength
		if i == newPiece.nSubPiece-1 {
			if newPiece.Len%downloader.subPieceLength != 0 {
				newPiece.SubPieceLen = newPiece.Len % downloader.subPieceLength
				println("newPiece.SubPieceLen")
				println(newPiece.SubPieceLen)
				fmt.Printf("pieceLen %v\n", newPiece.Len)

				if newPiece.Len != 1048576 {

				}
			}
		}
	}

	return newPiece

}