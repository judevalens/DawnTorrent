package PeerProtocol

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"time"
	"torrent/utils"

	//"github.com/emirpasic/gods/lists/arraylist"
	"github.com/emirpasic/gods/maps/hashmap"
	"math"
	"strconv"
	"sync"
	"torrent/parser"
)

type TorrentFile struct {
	announce        string
	announceList    []string
	comment         string
	createdBy       string
	creationDate    string
	encoding        string
	fileMode        int
	filesMutex      sync.RWMutex
	files           []fileInfo
	PiecesMutex     sync.RWMutex
	Pieces          []*Piece
	nPiece          int
	fileTotalLength int
	subPieceLen     int
	nSubPiece       int
	infoHash        string
	pieces          string
	pieceLength     int
	rootDirectory   string
	NeededPiece     *hashmap.Map
	currentPiece    *Piece
	behavior		string
	SelectNewPiece bool
	CurrentPieceIndex int
	torrent *Torrent
}

type fileInfo struct {
	path       string
	length     int
	fileIndex int
	startIndex int
	endIndex int

}



func initTorrentFile(torrent *Torrent, torrentPath string) *TorrentFile {
	file := new(TorrentFile)
	file.torrent = torrent
	metaInfo := parser.Unmarshall(torrentPath)
	file.infoHash = GetInfoHash(metaInfo)
	file.announce = metaInfo.MapString["announce"]
	file.announceList = make([]string,0)

	//TODO need fix
	/*for	_,v := range metaInfo.MapList["announce-list"].LList{
		for _ , s := range v.LString{
			file.announceList = append(file.announceList,s)
		}
	}*/
	file.creationDate = metaInfo.MapString["creation date"]
	file.encoding = metaInfo.MapString["encoding"]
	file.pieceLength, _ = strconv.Atoi(metaInfo.MapDict["info"].MapString["piece length"])


	file.NeededPiece = hashmap.New()
	_,isPresent := metaInfo.MapDict["info"].MapList["files"]

	if isPresent{
		file.fileMode = 1
		for fileIndex, v := range metaInfo.MapDict["info"].MapList["files"].LDict{
			filePath := v.MapList["path"].LString[0]
			fileLength, _ := strconv.Atoi(v.MapString["length"])

			newFile := newFileInfo(file,filePath,fileLength,fileIndex)
			file.files = append(file.files,newFile)
		}
	}else{
		file.fileMode = 0
		filePath := metaInfo.MapDict["info"].MapString["name"]
		fileLength, _ := strconv.Atoi(metaInfo.MapDict["info"].MapString["length"])

		file.files = append(file.files,newFileInfo(file,filePath,fileLength,0))
	}

	file.subPieceLen = int(math.Min(float64(file.pieceLength), SubPieceLen))
	file.nSubPiece = int(math.Ceil(float64(file.pieceLength)/float64(file.subPieceLen)))
	file.nPiece = int(math.Ceil(float64(file.fileTotalLength)/float64(file.pieceLength)))
	file.Pieces = make([]*Piece,file.nPiece)

	for i,_ := range file.Pieces {

		pieceLen := file.pieceLength
		if i == file.nPiece-1 {
			if file.fileTotalLength%file.pieceLength != 0 {
				pieceLen = file.fileTotalLength % file.pieceLength
				println("pieceLen")
				println(pieceLen)
				println(file.fileTotalLength)
			}
		}
		file.Pieces[i] = NewPiece(file,i,pieceLen)
		file.NeededPiece.Put(i,file.Pieces[i])
	}
	pieceIndex := 0
	for i:= 0; i < len(file.files);i++{
		f := file.files[i]
		pos := new(pos)
		pos.fileIndex = i
			for pos.end != f.endIndex{
				currentPiece := file.Pieces[pieceIndex]
				pos.start = int(math.Max(float64(currentPiece.pieceStartIndex),float64(f.startIndex)))
				pos.end= int(math.Min(float64(currentPiece.pieceEndIndex),float64(f.endIndex)))
				pos.writingIndex = pos.start-f.startIndex
				currentPiece.position = append(currentPiece.position,*pos)
				if currentPiece.pieceEndIndex == pos.end{
					pieceIndex++
				}

		}


	}

	file.behavior = "random"
	return file
}

func newFileInfo(torrentFile *TorrentFile,filePath string , fileLength int ,fileIndex int) fileInfo{
	f := fileInfo{}

	f.path = filePath
	f.length = fileLength

	torrentFile.fileTotalLength += f.length

	if fileIndex == 0 {
		f.startIndex = 0
	}else{
		f.startIndex = torrentFile.files[fileIndex-1].endIndex
	}

	f.endIndex = f.startIndex+f.length

	return f
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
// TODO clean up later
func NewPiece(torrentFile *TorrentFile,PieceIndex,pieceLength int) *Piece {
	newPiece := new(Piece)

	newPiece.PieceTotalLen = pieceLength
	newPiece.owners = make(map[string]*Peer)
	newPiece.Availability = 0
	newPiece.PieceIndex = PieceIndex
	newPiece.pieceStartIndex = PieceIndex*torrentFile.pieceLength
	newPiece.pieceEndIndex  = newPiece.pieceStartIndex+pieceLength
	newPiece.Pieces = make([]byte, newPiece.PieceTotalLen)
	newPiece.Status = "empty"
	newPiece.subPieceMask = make([]byte, int(math.Ceil(float64(torrentFile.nSubPiece)/float64(8))))
	newPiece.pendingRequestMutex = new(sync.RWMutex)
	newPiece.pendingRequest = hashmap.New()
	return newPiece

}
func (file *TorrentFile) AddSubPiece(msg MSG, peer *Peer) error {
	var err error = nil

	currentPiece := file.Pieces[msg.PieceIndex]
	subPieceIndex := int(math.Ceil(float64(msg.BeginIndex / SubPieceLen)))
	subPieceBitMaskIndex := int(math.Ceil(float64(subPieceIndex / 8)))
	subPieceBitIndex := subPieceIndex % 8
	file.PiecesMutex.Lock()
	isEmpty := utils.IsBitOn(currentPiece.subPieceMask[subPieceBitMaskIndex], subPieceBitIndex) == false
	file.PiecesMutex.Unlock()
	currentPiece.pendingRequestMutex.Lock()
	currentPiece.pendingRequest.Remove(msg.BeginIndex)
	currentPiece.pendingRequestMutex.Unlock()
	if isEmpty {
		fmt.Printf("Piece COunter %v\n",file.torrent.PieceCounter)
		file.torrent.PieceCounter++
		currentPiece.CurrentLen += msg.PieceLen
		println("msg.BeginIndex")
		println(msg.BeginIndex)
		copy(currentPiece.Pieces[msg.BeginIndex:msg.BeginIndex+msg.PieceLen], msg.Piece)
		file.PiecesMutex.Lock()
		currentPiece.subPieceMask[subPieceBitMaskIndex] = utils.BitMask(currentPiece.subPieceMask[subPieceBitMaskIndex], []int{subPieceBitIndex}, 1)
		file.PiecesMutex.Unlock()

		// piece is complete add it to the file
		isPieceComplete := currentPiece.PieceTotalLen == currentPiece.CurrentLen

		if isPieceComplete {
			file.NeededPiece.Remove(currentPiece.PieceIndex)
			currentPiece.Status = "complete"
			file.SelectNewPiece = true
			for _,pos := range currentPiece.position{
				_ = pos
				currentFile := file.files[pos.fileIndex]
				f, _ := os.OpenFile(utils.DawnTorrentHomeDir+"/"+currentFile.path, os.O_CREATE|os.O_RDWR, os.ModePerm)

				//getting the length of the piece
				pieceRelativeLen := pos.end-pos.start
				pieceRelativeStart := pos.start-(currentPiece.PieceIndex*file.pieceLength)
				pieceRelativeEnd := pieceRelativeStart+pieceRelativeLen
				_, _ = f.WriteAt(currentPiece.Pieces[pieceRelativeStart:pieceRelativeEnd], int64(pos.writingIndex))
			}

			currentPiece.Pieces = nil
			//os.Exit(2222234)
		} else {
			currentPiece.Status = "inProgress"
			file.SelectNewPiece = false
		}
	}

	//peer.numByteDownloaded += int(msg.PieceLen)
	//peer.lastTimeStamp = time.Now()
	//peer.time += time.Now().Sub(peer.lastTimeStamp).Seconds()
	//peer.DownloadRate = (float64(peer.numByteDownloaded)) / peer.time
	return err
}

type Piece struct {
	fileIndex int
	PieceTotalLen int
	CurrentLen    int
	SubPieceLen   int
	Status        string
	Pieces        []byte
	PieceIndex    int
	AbsIndex      int
	Availability  int
	owners        map[string]*Peer
	subPieceMask  []byte
	pieceStartIndex int
	pieceEndIndex int
	position []pos
	pendingRequestMutex *sync.RWMutex
	pendingRequest	*hashmap.Map
	requestFrom  *hashmap.Map
}

type pendingRequest struct {
	peerID	string
	startIndex int
	timeStamp	time.Time
	backUpPeers	*hashmap.Map
}

type pos struct {
	fileIndex int
	start int
	end int
	writingIndex int
}