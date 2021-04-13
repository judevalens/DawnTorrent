package protocol

import (
	"DawnTorrent/parser"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
)

type Torrent struct {
	Announce       string
	AnnounceList   []string
	Comment        string
	CreatedBy      string
	CreationDate   string
	Encoding       string
	piecesHash     string
	FileMode       int
	nPiece         int
	FileLength     int
	subPieceLength int
	nSubPiece      int
	InfoHash       string
	infoHashByte   [20]byte
	InfoHashHex    string
	pieceLength    int
	Name           string
	PieceHolder        []*byte
	PieceSelectionMode int
}

type TorrentFile struct {
	Path       string
	Length     int
	FileIndex  int
	StartIndex int
	EndIndex   int
}


func loadTorrentFile(filePath  string) *parser.Dict {
	return parser.Unmarshall(filePath)
}

func createNewTorrent(torrentMap *parser.Dict)  {
	downloader := new(Torrent)


	infoHashString, infoHashByte := GetInfoHash(torrentMap)
	downloader.InfoHash = infoHashString
	downloader.infoHashByte = infoHashByte
	downloader.InfoHashHex = hex.EncodeToString(downloader.infoHashByte[:])
	downloader.Announce = torrentMap.MapString["announce"]

	downloader.AnnounceList = make([]string, 0)
	downloader.AnnounceList = torrentMap.MapList["announce-list"].LString
	downloader.CreationDate = torrentMap.MapString["creation date"]
	downloader.Encoding = torrentMap.MapString["encoding"]
	downloader.piecesHash = torrentMap.MapDict["info"].MapString["pieces"]
	downloader.pieceLength, _ = strconv.Atoi(torrentMap.MapDict["info"].MapString["piece length"])

	_, isPresent := torrentMap.MapDict["info"].MapList["files"]
	fileProperties := make([]*TorrentFile, 0)
	downloader.Name = torrentMap.MapDict["info"].MapString["name"]
	totalLength := 0


	if isPresent {
		downloader.FileMode = 1
		for fileIndex, v := range torrentMap.MapDict["info"].MapList["files"].LDict {
			filePath := v.MapList["path"].LString[0]
			fileLength, _ := strconv.Atoi(v.MapString["length"])
			totalLength += fileLength
			newFile := createFileProperties(fileProperties, filePath, fileLength, fileIndex)
			fileProperties = append(fileProperties, newFile)
		}
	} else {
		downloader.FileMode = 0
		filePath := metaInfo.MapDict["info"].MapString["name"]
		fileLength, _ := strconv.Atoi(metaInfo.MapDict["info"].MapString["length"])
		totalLength += fileLength
		fileProperties = append(fileProperties, createFileProperties(fileProperties, filePath, fileLength, 0))
	}

	downloader.subPieceLength = SubPieceLen
	initFile(downloader, fileProperties, totalLength, totalLength, nil, StoppedState)

	downloader.PieceSelectionMode = sequentialSelection
	downloader.SelectNewPiece = true

	fmt.Printf("\n downloader Len %v , pieceLen %v, subPieceLength %v, n SubPieces %v \n", downloader.FileLength, downloader.pieceLength, downloader.subPieceLength, downloader.nSubPiece)
	return downloader
}

// store the path , len , start and end index of file
func createFileProperties(fileProperties []*TorrentFile, filePath string, fileLength int, fileIndex int) TorrentFile {
	torrent := TorrentFile{}

	torrent.Path = filePath
	torrent.Length = fileLength

	if fileIndex == 0 {
		torrent.StartIndex = 0
	} else {
		torrent.StartIndex = fileProperties[fileIndex-1].EndIndex
	}

	torrent.EndIndex = torrent.StartIndex + torrent.Length

	return torrent
}

// calculates the info hash based on pieces provided in the .torrent file
func GetInfoHash(dict *parser.Dict) (string, [20]byte) {

	// InnerStartingPosition leaves out the key

	startingPosition := dict.MapDict["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.MapDict["info"].KeyInfo.EndingPosition

	torrentFileString := parser.ToBencode(dict)

	torrentFileByte := []byte(torrentFileString)

	infoBytes := torrentFileByte[startingPosition:endingPosition]

	infoHash := sha1.Sum(infoBytes)
	infoHashSlice := infoHash[:]
	println(hex.EncodeToString(infoHashSlice))

	return string(infoHashSlice), infoHash

}