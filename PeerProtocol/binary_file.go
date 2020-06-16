package PeerProtocol

import (
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"io/ioutil"
	"log"
	"os"
	"time"
	//"github.com/emirpasic/gods/lists/arraylist"
	"math"
	"strconv"
	"sync"
)

const (
	completedRequest  = 2
	pendingRequest    = 1
	nonStartedRequest = 0
)

// loads .torrent file and process the metadata to start downloading and uploading
func initTorrentFile(torrent *Torrent, torrentPath string) *TorrentFile {
	file := new(TorrentFile)
	file.torrent = torrent
	metaInfo := parser.Unmarshall(torrentPath)

	infoHashString, infoHashByte := GetInfoHash(metaInfo)
	file.infoHash = infoHashString
	file.infoHashByte = infoHashByte
	file.Announce = metaInfo.MapString["announce"]

	file.AnnounceList = make([]string, 0)
	file.CreationDate = metaInfo.MapString["creation date"]
	file.Encoding = metaInfo.MapString["encoding"]
	file.piecesSha1Hash = metaInfo.MapDict["info"].MapString["pieces"]
	file.pieceLength, _ = strconv.Atoi(metaInfo.MapDict["info"].MapString["piece length"])
	_, isPresent := metaInfo.MapDict["info"].MapList["files"]
	fileInfos := make([]*fileInfo, 0)
	file.Name = metaInfo.MapDict["info"].MapString["name"]
	totalLength := 0
	if isPresent {
		file.FileMode = 1
		for fileIndex, v := range metaInfo.MapDict["info"].MapList["files"].LDict {
			filePath := v.MapList["path"].LString[0]
			fileLength, _ := strconv.Atoi(v.MapString["length"])
			totalLength += fileLength
			newFile := newFileInfo(fileInfos, filePath, fileLength, fileIndex)
			fileInfos = append(fileInfos, newFile)
		}
	} else {
		file.FileMode = 0

		filePath := metaInfo.MapDict["info"].MapString["name"]
		fileLength, _ := strconv.Atoi(metaInfo.MapDict["info"].MapString["length"])
		totalLength += fileLength
		fileInfos = append(fileInfos, newFileInfo(fileInfos, filePath, fileLength, 0))
	}
	file.subPieceLen = SubPieceLen
	initFile(file, fileInfos, totalLength, totalLength, nil, startedState)

	file.PieceSelectionBehavior = "random"
	file.SelectNewPiece = true

	file.saveTorrent()

	tp := utils.GetPath(utils.TorrentDataPath, file.Name)
	file2 := loadTorrent(tp)

	fmt.Printf("file 1 npIeces %v\n", file.nPiece)
	fmt.Printf("file 2 npIeces %v\n", file2.nPiece)
	fmt.Printf("file 1 infohash %v\n", file.infoHash)
	fmt.Printf("file 2 infohash %v\n", file2.infoHash)
	fmt.Printf("file 1 %v\n", len(file.piecesSha1Hash))
	fmt.Printf("file 2 %v\n", len(file2.piecesSha1Hash))
	fmt.Printf("file 1 pieces %v\n", len(file.Pieces))
	fmt.Printf("file 2 pieces %v\n", len(file2.Pieces))
	fmt.Printf("file 1 Status %v\n", file.status)
	fmt.Printf("file 2 Status %v\n", file2.status)

	return file
}

// saves the current state of a download, so it can be resumed later
func (file *TorrentFile) saveTorrent() {

	SavedTorrentData := new(SavedTorrentData)

	SavedTorrentData.Announce = file.Announce
	SavedTorrentData.InfoHash = file.infoHash
	SavedTorrentData.Comment = file.Comment
	SavedTorrentData.CreatedBy = file.CreatedBy
	SavedTorrentData.CreationDate = file.CreationDate
	SavedTorrentData.PieceLength = file.pieceLength
	SavedTorrentData.PiecesSha1 = file.piecesSha1Hash

	i := []byte(file.piecesSha1Hash)

	fmt.Printf("len sha1 %v", i)
	SavedTorrentData.Left = file.left
	SavedTorrentData.FileLen = file.FileLen
	SavedTorrentData.SubPieceLen = file.subPieceLen
	SavedTorrentData.Name = file.Name
	SavedTorrentData.Status = file.status

	pieces := make([]*Piece, file.nPiece)

	for i, p := range file.Pieces {
		newPiece := new(Piece)

		file.PiecesMutex.Lock()
		newPiece.Status = p.Status
		newPiece.PieceIndex = p.PieceIndex
		file.PiecesMutex.Unlock()
		pieces[i] = newPiece
	}

	SavedTorrentData.FileInfos = file.files

	SavedTorrentData.Pieces = pieces

	SavedTorrentDataBuffer := bytes.NewBuffer(make([]byte, 0))
	SavedTorrentDataJSONEncoder := json.NewEncoder(SavedTorrentDataBuffer)
	SavedTorrentDataJSONEncoder.SetEscapeHTML(false)
	SavedTorrentDataJSONEncoder.SetIndent("", "")
	_ = SavedTorrentDataJSONEncoder.Encode(SavedTorrentData)

	pieceHashPath := utils.GetPath(utils.PieceHashPath, file.Name)

	if _, err := os.Stat(pieceHashPath); os.IsNotExist(err) {
		_ = ioutil.WriteFile(pieceHashPath, []byte(file.piecesSha1Hash), os.ModePerm)
	}

	savedTorrentDataPath := utils.GetPath(utils.TorrentDataPath, file.Name)
	wErr := ioutil.WriteFile(savedTorrentDataPath, SavedTorrentDataBuffer.Bytes(), os.ModePerm)
	if wErr != nil {
		fmt.Printf("%v", wErr)
		os.Exit(222)
	}

}

// loads a saved torrent
func loadTorrent(torrentPath string) *TorrentFile {
	torrentDataJSON, errr := ioutil.ReadFile(torrentPath)

	if errr != nil {
		log.Fatal(errr)
	}

	println(torrentDataJSON)

	torrentData := new(SavedTorrentData)
	_ = json.Unmarshal(torrentDataJSON, torrentData)

	file := new(TorrentFile)

	file.infoHash = torrentData.InfoHash
	file.Announce = torrentData.Announce
	file.AnnounceList = make([]string, 0)
	file.CreationDate = torrentData.CreationDate
	file.CreatedBy = torrentData.CreatedBy

	fmt.Printf("\nfileLen  %v pieceLen %v\n", torrentData.FileLen, torrentData.PieceLength)

	file.pieceLength = torrentData.PieceLength
	file.left = torrentData.Left
	file.Name = torrentData.Name
	file.subPieceLen = torrentData.SubPieceLen

	pieceHashPath := utils.GetPath(utils.PieceHashPath, file.Name)
	pieceHash, _ := ioutil.ReadFile(pieceHashPath)
	file.piecesSha1Hash = string(pieceHash)

	// will create the pieces
	initFile(file, torrentData.FileInfos, torrentData.FileLen, torrentData.Left, torrentData.Pieces, torrentData.Status)

	return file
}

// initialize pieces of file
func initFile(file *TorrentFile, fileInfos []*fileInfo, fileTotalLength, left int, pieces []*Piece, torrentStatus int) {
	file.status = torrentStatus
	file.NeededPiece = make(map[int]*Piece)

	for _, f := range fileInfos {
		file.files = append(file.files, f)
	}

	file.FileLen = fileTotalLength
	file.left = left
	file.subPieceLen = int(math.Min(float64(file.pieceLength), float64(file.subPieceLen)))
	file.nSubPiece = int(math.Ceil(float64(file.pieceLength) / float64(file.subPieceLen)))
	file.nPiece = int(math.Ceil(float64(file.FileLen) / float64(file.pieceLength)))
	fmt.Printf("nPieces %v", file.nPiece)

	file.Pieces = make([]*Piece, file.nPiece)
	file.PieceAvailability = arraylist.New()
	file.pieceAvailabilityMutex = new(sync.RWMutex)

	for i, _ := range file.Pieces {

		pieceLen := file.pieceLength
		if i == file.nPiece-1 {
			if file.FileLen%file.pieceLength != 0 {
				pieceLen = file.FileLen % file.pieceLength
				println("pieceLen")
				println(pieceLen)
				println(file.FileLen)

			}
		}

		thisPieceStatus := EmptyPiece

		if pieces != nil {
			thisPieceStatus = pieces[i].Status
		}
		file.Pieces[i] = NewPiece(file, i, pieceLen, thisPieceStatus)
		//file.Pieces[i].Availability = i

		if thisPieceStatus != CompletePiece {
			file.NeededPiece[i] = file.Pieces[i]
		}

		file.PieceAvailability.Add(file.Pieces[i])
	}

	pieceIndex := 0

	// process overlapping pieces
	// determine which subPiece belongs to a file and the position where it starts
	for i := 0; i < len(file.files); i++ {
		f := file.files[i]
		pos := new(pos)
		pos.fileIndex = i
		for pos.end != f.EndIndex {
			currentPiece := file.Pieces[pieceIndex]
			pos.start = int(math.Max(float64(currentPiece.pieceStartIndex), float64(f.StartIndex)))
			pos.end = int(math.Min(float64(currentPiece.pieceEndIndex), float64(f.EndIndex)))
			pos.writingIndex = pos.start - f.StartIndex
			currentPiece.position = append(currentPiece.position, *pos)
			if currentPiece.pieceEndIndex == pos.end {
				pieceIndex++
			}

		}

	}
}

// store the path , len , start and end index of file
func newFileInfo(fileInfos []*fileInfo, filePath string, fileLength int, fileIndex int) *fileInfo {
	f := new(fileInfo)

	f.Path = filePath
	f.Length = fileLength

	if fileIndex == 0 {
		f.StartIndex = 0
	} else {
		f.StartIndex = fileInfos[fileIndex-1].EndIndex
	}

	f.EndIndex = f.StartIndex + f.Length

	return f
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
	//	println(hex.EncodeToString(bSlice))

	return string(infoHashSlice), infoHash

}

//	Creates a Piece object and initialize subPieceRequest for this piece
func NewPiece(torrentFile *TorrentFile, PieceIndex, pieceLength int, status int) *Piece {
	newPiece := new(Piece)

	newPiece.Len = pieceLength
	newPiece.owners = make(map[string]*Peer)
	newPiece.Availability = 0
	newPiece.PieceIndex = PieceIndex
	newPiece.pieceStartIndex = PieceIndex * torrentFile.pieceLength
	newPiece.pieceEndIndex = newPiece.pieceStartIndex + newPiece.Len
	newPiece.Pieces = make([]byte, newPiece.Len)
	newPiece.Status = status
	newPiece.subPieceMask = make([]byte, int(math.Ceil(float64(torrentFile.nSubPiece)/float64(8))))
	newPiece.pendingRequestMutex = new(sync.RWMutex)
	newPiece.pendingRequest = make([]*PieceRequest, 0)

	newPiece.nSubPiece = int(math.Ceil(float64(pieceLength) / float64(torrentFile.subPieceLen)))
	newPiece.neededSubPiece = make([]*PieceRequest, newPiece.nSubPiece)
	for i := 0; i < newPiece.nSubPiece; i++ {
		newPiece.SubPieceLen = torrentFile.subPieceLen
		if i == newPiece.nSubPiece-1 {
			if newPiece.Len%torrentFile.subPieceLen != 0 {
				newPiece.SubPieceLen = newPiece.Len % torrentFile.subPieceLen
				println("newPiece.SubPieceLen")
				println(newPiece.SubPieceLen)
				fmt.Printf("pieceLen %v\n", newPiece.Len)

				if newPiece.Len != 1048576 {

					//	os.Exit(2828)

				}
			}
		}

		pendingRequest := new(PieceRequest)

		pendingRequest.startIndex = i * SubPieceLen

		msg := MSG{MsgID: RequestMsg, PieceIndex: newPiece.PieceIndex, BeginIndex: pendingRequest.startIndex, PieceLen: newPiece.SubPieceLen}

		pendingRequest.msg = GetMsg(msg, nil)
		pendingRequest.status = nonStartedRequest
		pendingRequest.subPieceIndex = i
		newPiece.neededSubPiece[i] = pendingRequest
	}

	return newPiece

}

// used to sort pieces by availability
func pieceAvailabilityComparator(a, b interface{}) int {
	pieceA := a.(*Piece)
	pieceB := b.(*Piece)

	switch {
	case pieceA.Availability > pieceB.Availability:
		return 1
	case pieceA.Availability < pieceB.Availability:
		return -1
	default:
		return 0
	}
}

// sort pieces by availability
func (file *TorrentFile) SortPieceByAvailability() {
	file.pieceAvailabilityMutex.Lock()
	if time.Now().Sub(file.PieceAvailabilityTimeStamp) >= time.Second*5 {
		file.PieceAvailability.Sort(pieceAvailabilityComparator)
		file.PieceAvailabilityTimeStamp = time.Now()
	}
	file.pieceAvailabilityMutex.Unlock()
}

// assemble and write the pieces downloaded
func (file *TorrentFile) AddSubPiece(msg *MSG, peer *Peer) error {
	var err error = nil


	// when a peer send a requested piece , the pending request object is removed from the peer pending request list
	msg.Peer.peerPendingRequestMutex.Lock()
	nPendingRequest := len(msg.Peer.peerPendingRequest)
	for i := nPendingRequest; i >= 0; i++ {

		if i < len(msg.Peer.peerPendingRequest) {
			pendingRequest := msg.Peer.peerPendingRequest[i]

			if pendingRequest.startIndex == msg.BeginIndex {
				pendingRequest.status = completedRequest
				msg.Peer.peerPendingRequest[i], msg.Peer.peerPendingRequest[len(msg.Peer.peerPendingRequest)-1] = msg.Peer.peerPendingRequest[len(peer.peerPendingRequest)-1], nil
				msg.Peer.peerPendingRequest = msg.Peer.peerPendingRequest[:len(msg.Peer.peerPendingRequest)-1]
				break
			}
		} else {
			break
		}
	}
	msg.Peer.peerPendingRequestMutex.Unlock()

	// determines which piece this subPiece belongs to
	currentPiece := file.Pieces[msg.PieceIndex]
	subPieceIndex := int(math.Ceil(float64(msg.BeginIndex / SubPieceLen)))
	subPieceBitMaskIndex := int(math.Ceil(float64(subPieceIndex / 8)))
	subPieceBitIndex := subPieceIndex % 8
	file.PiecesMutex.Lock()
	isEmpty := utils.IsBitOn(currentPiece.subPieceMask[subPieceBitMaskIndex], subPieceBitIndex) == false
	currentPiece.subPieceMask[subPieceBitMaskIndex] = utils.BitMask(currentPiece.subPieceMask[subPieceBitMaskIndex], []int{subPieceBitIndex}, 1)
	file.PiecesMutex.Unlock()

	// if we haven't already receive this piece , we saved it
	if isEmpty {
		if msg.PieceIndex != file.CurrentPieceIndex {
			fmt.Printf("msg Piece Index %v current Piece Index %v", msg.PieceIndex, file.CurrentPieceIndex)
			os.Exit(223444)
		}

		currentPiece.pendingRequestMutex.Lock()
		for i := len(currentPiece.pendingRequest) - 1; i >= 0; i-- {

			if msg.BeginIndex == currentPiece.pendingRequest[i].startIndex {
				currentPiece.pendingRequest[i], currentPiece.pendingRequest[len(currentPiece.pendingRequest)-1] = currentPiece.pendingRequest[len(currentPiece.pendingRequest)-1], nil

				currentPiece.pendingRequest = currentPiece.pendingRequest[:len(currentPiece.pendingRequest)-1]

				break
			}
		}
		currentPiece.pendingRequestMutex.Unlock()

		///fmt.Printf("Piece COunter %v\n",file.torrent.PieceCounter)
		file.torrent.PieceCounter++
		currentPiece.CurrentLen += len(msg.Piece)
		file.totalDownloaded += msg.PieceLen
		file.left -= msg.PieceLen
		fmt.Printf("downloaded %v mb in %v \n", file.totalDownloaded/1000000.0, time.Now().Sub(file.timeS))

		copy(currentPiece.Pieces[msg.BeginIndex:msg.BeginIndex+msg.PieceLen], msg.Piece)

		// piece is complete add it to the file
		file.PiecesMutex.Lock()
		isPieceComplete := currentPiece.Len == currentPiece.CurrentLen
		file.PiecesMutex.Unlock()

		// once a piece is complete we write the it to file
		if isPieceComplete {
			delete(file.NeededPiece, currentPiece.PieceIndex)
			currentPiece.Status = CompletePiece
			file.SelectNewPiece = true
			//	fmt.Printf("piece %v is complete\n",currentPiece.PieceIndex)

			//	a piece can contains data that goes into multiple files
			// pos struct contains start and end index , that indicates into which file a particular slice of data goes
			for _, pos := range currentPiece.position {
				currentFile := file.files[pos.fileIndex]

				if _, err := os.Stat(utils.TorrentHomeDir); os.IsNotExist(err) {
					_ = os.MkdirAll(utils.TorrentHomeDir, os.ModePerm)
				}

				f, _ := os.OpenFile(utils.TorrentHomeDir+"/"+currentFile.Path, os.O_CREATE|os.O_RDWR, os.ModePerm)

				//getting the length of the piece
				pieceRelativeLen := pos.end - pos.start
				pieceRelativeStart := pos.start - (currentPiece.PieceIndex * file.pieceLength)
				pieceRelativeEnd := pieceRelativeStart + pieceRelativeLen
				_, _ = f.WriteAt(currentPiece.Pieces[pieceRelativeStart:pieceRelativeEnd], int64(pos.writingIndex))
			}

			currentPiece.Pieces = make([]byte, 0)
		} else {
			//	fmt.Printf("not complete yet , %v/%v pieceindex : %v\n",currentPiece.Len,currentPiece.CurrentLen,currentPiece.PieceIndex)
			currentPiece.Status = InProgressPiece
			file.SelectNewPiece = false
		}
	}


	/// aggregate stats about a peer
	// will be used to determine which peer to unchoke
	// and which that we will prefer when requesting subPieces
	if time.Now().Sub(peer.lastTimeStamp).Seconds() >= peerDownloadRatePeriod.Seconds() {
		d := time.Now().Sub(peer.lastTimeStamp).Seconds()
		peer.DownloadRate = float64(peer.numByteDownloaded) / d
		peer.numByteDownloaded = msg.PieceLen
		peer.lastTimeStamp = time.Now()

	} else {
		peer.numByteDownloaded += msg.PieceLen
	}

	fmt.Printf("peer: %v , download rate: %v byte/s \n", msg.Peer.ip, peer.DownloadRate)

	return err
}
