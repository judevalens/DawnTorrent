package app

import (
	"DawnTorrent/parser"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strconv"
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
	InfoHashByte   [20]byte
	InfoHashHex    string
	InfoHash       string
	pieceLength    int
	Name           string
	bitfield       []byte
}

type fileMetadata struct {
	Path       string
	Length     int
	FileIndex  int
	StartIndex int
	EndIndex   int
}

func CreateNewTorrent(torrentPath string) (*Torrent, error) {
	torrentMap, err := parser.Unmarshall(torrentPath)
	if err != nil {
		return nil, err
	}
	torrent := new(Torrent)

	infoHashByte, hexInfoHash := GetInfoHash(torrentMap)

	torrent.InfoHashByte = infoHashByte
	torrent.InfoHashHex = hexInfoHash
	torrent.InfoHash = string(torrent.InfoHashByte[:])
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
	// initializing bitfield


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


func getRequestId(pieceIndex,startIndex int) string  {
	pieceIndex2 := strconv.Itoa(pieceIndex)
	startIndex2 := strconv.Itoa(startIndex)

	return pieceIndex2+"-"+startIndex2
}
