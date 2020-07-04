package PeerProtocol

import (
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"io/ioutil"
	"log"
	"os"
	"sync/atomic"
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

	InitTorrentFile_   = 0
	ResumeTorrentFile_ = 1
)

// loads .torrent file and process the metadata to start downloading and uploading
func InitTorrentFile(torrent *Torrent, torrentPath string) *TorrentFile {
	file := new(TorrentFile)
	file.torrent = torrent
	file.downloadRateMutex = new(sync.Mutex)
	file.addPieceChannel = make(chan *MSG,5)
	metaInfo := parser.Unmarshall(torrentPath)

	infoHashString, infoHashByte := GetInfoHash(metaInfo)
	file.InfoHash = infoHashString
	file.infoHashByte = infoHashByte
	file.InfoHashHex = hex.EncodeToString(file.infoHashByte[:20])
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
	initFile(file, fileInfos, totalLength, totalLength, nil, StoppedState)

	file.PieceSelectionBehavior = "random"
	file.SelectNewPiece = true

	fmt.Printf("\n file Len %v , pieceLen %v, subPieceLen %v, n SubPieces %v \n",file.FileLen,file.pieceLength,file.subPieceLen,file.nSubPiece)

	go file.fileAssembler()
	return file
}

func (file *TorrentFile) setState(state int) {
	atomic.StoreInt32(file.Status, int32(state))
}

// resume a saved torrentFile
func resumeTorrentFile(path string) *TorrentFile {

	torrentDataJSON, errr := ioutil.ReadFile(path)

	if errr != nil {
		log.Fatal(errr)
	}

	println(torrentDataJSON)

	torrentData := new(SavedTorrentData)
	_ = json.Unmarshal(torrentDataJSON, torrentData)

	file := new(TorrentFile)

	file.InfoHash = torrentData.InfoHash
	file.Announce = torrentData.Announce
	file.AnnounceList = make([]string, 0)
	file.CreationDate = torrentData.CreationDate
	file.CreatedBy = torrentData.CreatedBy

	file.pieceLength = torrentData.PieceLength
	file.left = torrentData.Left
	file.Name = torrentData.Name
	file.subPieceLen = torrentData.SubPieceLen
	pieceHashPath := utils.GetPath(utils.PieceHashPath, file.Name)
	pieceHash, _ := ioutil.ReadFile(pieceHashPath)
	file.piecesSha1Hash = string(pieceHash)
	//will create the pieces
	initFile(file, torrentData.FileInfos, torrentData.FileLen, torrentData.Left, torrentData.Pieces, torrentData.State)
	fmt.Printf("\nfile info\n file name : %v \n fileLen  %v \n pieceLen %v\n subPieceLen %v\n infoHash %v\n", file.Name, file.FileLen, file.pieceLength, file.subPieceLen, file.InfoHash)

	return file
}

// saves the current state of a download, so it can be resumed later
func (file *TorrentFile) saveTorrent() {
	SavedTorrentData := new(SavedTorrentData)
	SavedTorrentData.Announce = file.Announce
	SavedTorrentData.InfoHash = file.InfoHash
	SavedTorrentData.Comment = file.Comment
	SavedTorrentData.CreatedBy = file.CreatedBy
	SavedTorrentData.CreationDate = file.CreationDate
	SavedTorrentData.PieceLength = file.pieceLength
	///	SavedTorrentData.PiecesSha1 = file.piecesSha1Hash

	SavedTorrentData.Left = file.left
	SavedTorrentData.FileLen = file.FileLen
	SavedTorrentData.SubPieceLen = file.subPieceLen
	SavedTorrentData.Name = file.Name
	SavedTorrentData.State = int(atomic.LoadInt32(file.Status))

	pieces := make([]*Piece, file.nPiece)

	for i, p := range file.Pieces {
		newPiece := new(Piece)

		newPiece.State = p.State
		newPiece.PieceIndex = p.PieceIndex
		pieces[i] = newPiece
	}

	SavedTorrentData.FileInfos = file.Files

	SavedTorrentData.Pieces = pieces

	SavedTorrentDataBuffer := bytes.NewBuffer(make([]byte, 0))
	SavedTorrentDataJSONEncoder := json.NewEncoder(SavedTorrentDataBuffer)
	SavedTorrentDataJSONEncoder.SetEscapeHTML(false)
	SavedTorrentDataJSONEncoder.SetIndent("", "")
	_ = SavedTorrentDataJSONEncoder.Encode(SavedTorrentData)

	//pieceHashPath := utils.GetPath(utils.PieceHashPath, file.Name)

	//if _, err := os.Stat(pieceHashPath); os.IsNotExist(err) {
	//	writeErr = ioutil.WriteFile(pieceHashPath, []byte(file.piecesSha1Hash), os.ModePerm)
	//}

	/*savedTorrentDataPath := utils.GetPath(utils.TorrentDataPath, file.Name)
	//wErr := ioutil.WriteFile(savedTorrentDataPath, SavedTorrentDataBuffer.Bytes(), os.ModePerm)
	if wErr != nil {
		fmt.Printf("%v", wErr)
		//os.Exit(222)
	}
	*/
}
func (file *TorrentFile) saveTorrentForUiUpdate() {
	SavedTorrentData := new(SavedTorrentData)
	SavedTorrentData.Announce = file.Announce
	SavedTorrentData.InfoHash = file.InfoHash
	SavedTorrentData.Comment = file.Comment
	SavedTorrentData.CreatedBy = file.CreatedBy
	SavedTorrentData.CreationDate = file.CreationDate
	SavedTorrentData.PieceLength = file.pieceLength
	///	SavedTorrentData.PiecesSha1 = file.piecesSha1Hash

	i := []byte(file.piecesSha1Hash)

	fmt.Printf("len sha1 %v", i)
	SavedTorrentData.Left = file.left
	SavedTorrentData.FileLen = file.FileLen
	SavedTorrentData.SubPieceLen = file.subPieceLen
	SavedTorrentData.Name = file.Name
	SavedTorrentData.State = int(atomic.LoadInt32(file.Status))

	pieces := make([]*Piece, file.nPiece)

	for i, p := range file.Pieces {
		newPiece := new(Piece)

		file.PiecesMutex.Lock()
		newPiece.State = p.State
		newPiece.PieceIndex = p.PieceIndex
		file.PiecesMutex.Unlock()
		pieces[i] = newPiece
	}

	SavedTorrentData.FileInfos = file.Files

	SavedTorrentData.Pieces = pieces

	SavedTorrentDataBuffer := bytes.NewBuffer(make([]byte, 0))
	SavedTorrentDataJSONEncoder := json.NewEncoder(SavedTorrentDataBuffer)
	SavedTorrentDataJSONEncoder.SetEscapeHTML(false)
	SavedTorrentDataJSONEncoder.SetIndent("", "")
	_ = SavedTorrentDataJSONEncoder.Encode(SavedTorrentData)
}

// initialize pieces of file
func initFile(file *TorrentFile, fileInfos []*fileInfo, fileTotalLength, left int, pieces []*Piece, torrentStatus int) {
	file.Status = new(int32)
	atomic.StoreInt32(file.Status, int32(torrentStatus))
	file.NeededPiece = make(map[int]*Piece)

	for _, f := range fileInfos {
		file.Files = append(file.Files, f)
	}

	file.FileLen = fileTotalLength
	file.left = left
	file.subPieceLen = int(math.Min(float64(file.pieceLength), float64(file.subPieceLen)))
	file.nSubPiece = int(math.Ceil(float64(file.pieceLength) / float64(file.subPieceLen)))
	file.nPiece = int(math.Ceil(float64(file.FileLen) / float64(file.pieceLength)))
	fmt.Printf("nPieces %v\n", file.nPiece)

	file.Pieces = make([]*Piece, file.nPiece)
	file.PieceAvailability = arraylist.New()
	file.pieceAvailabilityMutex = new(sync.RWMutex)

	for i := range file.Pieces {

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
			thisPieceStatus = pieces[i].State
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
	for i := 0; i < len(file.Files); i++ {
		f := file.Files[i]
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
	println(hex.EncodeToString(infoHashSlice))

	return string(infoHashSlice), infoHash

}

func (file *TorrentFile) isPieceValid(piece *Piece)bool{

	validHash := file.piecesSha1Hash[piece.PieceIndex*20:(piece.PieceIndex*20)+20]

	fmt.Printf("piece Index %v\n",piece.PieceIndex)

	pieceHash := sha1.Sum(piece.Pieces)
	pieceHashSlice := pieceHash[:]

	fmt.Printf("\n pieceHash :\n")
	fmt.Printf(" %v\n",string(pieceHashSlice))
	fmt.Printf("\n# neededSubpiece : %v\n",len(piece.neededSubPiece))
	fmt.Printf("\n# PieceCounter : %v\n",file.torrent.PieceCounter)
	fmt.Printf("\n currentValidHash :\n")
	fmt.Printf(" %v\n",validHash)
	fmt.Printf("Piece are equal : %v\n", validHash == string(pieceHashSlice))


	if !( validHash == string(pieceHashSlice)){
		os.Exit(223)
	}

	return validHash == string(pieceHashSlice)
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
	newPiece.State = status
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

	file.PiecesMutex.Lock()
	if atomic.LoadInt32(file.Status) == int32(StartedState) {
		// determines which piece this subPiece belongs to
		currentPiece := file.Pieces[msg.PieceIndex]
		subPieceIndex := int(math.Ceil(float64(msg.BeginIndex / SubPieceLen)))
		subPieceBitMaskIndex := int(math.Ceil(float64(subPieceIndex / 8)))
		subPieceBitIndex := subPieceIndex % 8

		if subPieceBitMaskIndex < len(currentPiece.subPieceMask) {
			// when a peer send a requested piece , the pending request object is removed from the peer pending request list
			msg.Peer.mutex.Lock()
			nPendingRequest := len(msg.Peer.peerPendingRequest)
			for i := nPendingRequest - 1; i >= 0; i-- {
				if i < len(msg.Peer.peerPendingRequest) {
					pendingRequest := msg.Peer.peerPendingRequest[i]

					if pendingRequest.startIndex == msg.BeginIndex {
						pendingRequest.status = completedRequest
						msg.Peer.peerPendingRequest[i], msg.Peer.peerPendingRequest[len(msg.Peer.peerPendingRequest)-1] = msg.Peer.peerPendingRequest[len(peer.peerPendingRequest)-1], nil
						msg.Peer.peerPendingRequest = msg.Peer.peerPendingRequest[:len(msg.Peer.peerPendingRequest)-1]
						break
					}
				}
			}
			msg.Peer.mutex.Unlock()
			isEmpty := utils.IsBitOn(currentPiece.subPieceMask[subPieceBitMaskIndex], subPieceBitIndex) == false
			currentPiece.subPieceMask[subPieceBitMaskIndex] = utils.BitMask(currentPiece.subPieceMask[subPieceBitMaskIndex], []int{subPieceBitIndex}, 1)

			// if we haven't already receive this piece , we saved it
			if isEmpty {
				if msg.PieceIndex != file.CurrentPieceIndex {
					fmt.Printf("msg Piece App %v current Piece App %v", msg.PieceIndex, file.CurrentPieceIndex)
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
				file.TotalDownloaded += msg.PieceLen
				file.left -= msg.PieceLen
				fmt.Printf("downloaded %v mb in %v \n", file.TotalDownloaded/1000000.0, time.Now().Sub(file.timeS))

				copy(currentPiece.Pieces[msg.BeginIndex:msg.BeginIndex+msg.PieceLen], msg.Piece)

				fmt.Printf("\nbegin Index : %v, end index : %v \n",msg.BeginIndex,msg.BeginIndex+msg.PieceLen)

				// piece is complete add it to the file
				isPieceComplete := currentPiece.Len == currentPiece.CurrentLen

				// once a piece is complete we write the it to file
				if isPieceComplete {
					delete(file.NeededPiece, currentPiece.PieceIndex)
					currentPiece.State = CompletePiece
					//file.SelectNewPiece = true
					//	fmt.Printf("piece %v is complete\n",currentPiece.PieceIndex)

					//	a piece can contains data that goes into multiple files
					// pos struct contains start and end index , that indicates into which file a particular slice of data goes
					for _, pos := range currentPiece.position {
						currentFile := file.Files[pos.fileIndex]

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
					currentPiece.State = InProgressPiece
					file.SelectNewPiece = false
				}


				if time.Now().Sub(peer.lastTimeStamp).Seconds() >= peerDownloadRatePeriod.Seconds() {
					d := time.Now().Sub(peer.lastTimeStamp).Seconds()
					peer.numByteDownloaded += msg.PieceLen
					peer.DownloadRate = float64(peer.numByteDownloaded) / d
					peer.numByteDownloaded = msg.PieceLen
					peer.lastTimeStamp = time.Now()

				} else {
					peer.numByteDownloaded += msg.PieceLen
				}

			}

		}

	}
	file.PiecesMutex.Unlock()

	file.downloadRateMutex.Lock()
	file.tempDownloadCounter += len(msg.Piece)
	if time.Now().Sub(file.downloadRateTimeStamp) > time.Second*3 {
		duration := time.Now().Sub(file.downloadRateTimeStamp)
		file.DownloadRate = float64(file.tempDownloadCounter) / duration.Seconds()
		file.downloadRateTimeStamp = time.Now()
		file.tempDownloadCounter = 0
	}
	file.downloadRateMutex.Unlock()
	/// aggregate stats about a peer
	// will be used to determine which peer to unchoke
	// and which that we will prefer when requesting subPieces

	//	fmt.Printf("peer: %v , download rate: %v byte/s \n", msg.Peer.ip, peer.DownloadRate)

	return err
}
func (file *TorrentFile) fileAssembler() error {
	var err error = nil


	for {
		msg := <- file.addPieceChannel
		if atomic.LoadInt32(file.Status) == int32(StartedState) {
			currentPiece := file.Pieces[msg.PieceIndex]
			subPieceIndex := int(math.Ceil(float64(msg.BeginIndex / SubPieceLen)))
			subPieceBitMaskIndex := int(math.Ceil(float64(subPieceIndex / 8)))
			subPieceBitIndex := subPieceIndex % 8
			file.torrent.PieceCounter++

			if subPieceBitMaskIndex < len(currentPiece.subPieceMask) {

				// when a peer send a requested piece , the pending request object is removed from the peer pending request list
				msg.Peer.mutex.Lock()
				nPendingRequest := len(msg.Peer.peerPendingRequest)
				for i := nPendingRequest - 1; i >= 0; i-- {
					if i < len(msg.Peer.peerPendingRequest) {
						pendingRequest := msg.Peer.peerPendingRequest[i]

							pendingRequest.status = completedRequest
							msg.Peer.peerPendingRequest[i], msg.Peer.peerPendingRequest[len(msg.Peer.peerPendingRequest)-1] = msg.Peer.peerPendingRequest[len(msg.Peer.peerPendingRequest)-1], nil
							msg.Peer.peerPendingRequest = msg.Peer.peerPendingRequest[:len(msg.Peer.peerPendingRequest)-1]
							break
						}
					}
				}
				msg.Peer.mutex.Unlock()
				isEmpty := utils.IsBitOn(currentPiece.subPieceMask[subPieceBitMaskIndex], subPieceBitIndex) == false
				currentPiece.subPieceMask[subPieceBitMaskIndex] = utils.BitMask(currentPiece.subPieceMask[subPieceBitMaskIndex], []int{subPieceBitIndex}, 1)

				// if we haven't already receive this piece , we saved it
				if isEmpty {
					if msg.PieceIndex != file.CurrentPieceIndex {
						fmt.Printf("msg Piece App %v current Piece App %v", msg.PieceIndex, file.CurrentPieceIndex)
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
					currentPiece.CurrentLen += len(msg.Piece)
					file.TotalDownloaded += msg.PieceLen
					file.left -= msg.PieceLen
					///fmt.Printf("downloaded %v mb in %v \n", file.TotalDownloaded/1000000.0, time.Now().Sub(file.timeS))

					ncopied := copy(currentPiece.Pieces[msg.BeginIndex:msg.BeginIndex+msg.PieceLen], msg.Piece)

					if ncopied != 16384 {
						os.Exit(23)
					}
					fmt.Printf("n copied is : %v\n", ncopied)
					fmt.Printf("begin Index : %v, end index : %v \n",msg.BeginIndex,msg.BeginIndex+msg.PieceLen)

					// piece is complete add it to the file
					isPieceComplete := currentPiece.Len == currentPiece.CurrentLen

					// once a piece is complete we write the it to file
					if isPieceComplete {
						fmt.Printf(" currentPiece.Len %v, currentPiece.CurrentLen %v\n",currentPiece.Len,currentPiece.CurrentLen)
						file.isPieceValid(currentPiece)
						delete(file.NeededPiece, currentPiece.PieceIndex)
						currentPiece.State = CompletePiece
						//file.SelectNewPiece = true
						//	fmt.Printf("piece %v is complete\n",currentPiece.PieceIndex)

						//	a piece can contains data that goes into multiple files
						// pos struct contains start and end index , that indicates into which file a particular slice of data goes
						for _, pos := range currentPiece.position {
							currentFile := file.Files[pos.fileIndex]

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
						currentPiece.State = InProgressPiece
						file.SelectNewPiece = false
					}


					if time.Now().Sub(msg.Peer.lastTimeStamp).Seconds() >= peerDownloadRatePeriod.Seconds() {
						d := time.Now().Sub(msg.Peer.lastTimeStamp).Seconds()
						msg.Peer.numByteDownloaded += msg.PieceLen
						msg.Peer.DownloadRate = float64(msg.Peer.numByteDownloaded) / d
						msg.Peer.numByteDownloaded = msg.PieceLen
						msg.Peer.lastTimeStamp = time.Now()

					} else {
						msg.Peer.numByteDownloaded += msg.PieceLen
					}

				}

			file.downloadRateMutex.Lock()
			file.tempDownloadCounter += len(msg.Piece)
			if time.Now().Sub(file.downloadRateTimeStamp) > time.Second*3 {
				duration := time.Now().Sub(file.downloadRateTimeStamp)
				file.DownloadRate = float64(file.tempDownloadCounter) / duration.Seconds()
				file.downloadRateTimeStamp = time.Now()
				file.tempDownloadCounter = 0
			}
			file.downloadRateMutex.Unlock()

			}


		}









	/// aggregate stats about a peer
	// will be used to determine which peer to unchoke
	// and which that we will prefer when requesting subPieces

	//	fmt.Printf("peer: %v , download rate: %v byte/s \n", msg.Peer.ip, peer.DownloadRate)

	return err
}