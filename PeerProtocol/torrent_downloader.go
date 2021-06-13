package PeerProtocol

import (
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"sort"
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
func initDownloader(torrentPath string) *TorrentDownloader {
	downloader := new(TorrentDownloader)

	metaInfo, _ := parser.Unmarshall(torrentPath)

	infoHashString, infoHashByte := GetInfoHash(metaInfo)
	downloader.InfoHash = infoHashString
	downloader.infoHashByte = infoHashByte
	downloader.InfoHashHex = hex.EncodeToString(downloader.infoHashByte[:])
	downloader.Announce = metaInfo.Strings["announce"]

	downloader.AnnounceList = make([]string, 0)
	downloader.CreationDate = metaInfo.Strings["creation date"]
	downloader.Encoding = metaInfo.Strings["encoding"]
	downloader.piecesSha1Hash = metaInfo.BMaps["info"].Strings["pieces"]
	downloader.pieceLength, _ = strconv.Atoi(metaInfo.BMaps["info"].Strings["piece length"])
	_, isPresent := metaInfo.BMaps["info"].BLists["files"]
	fileProperties := make([]*fileProperty, 0)
	downloader.Name = metaInfo.BMaps["info"].Strings["name"]
	totalLength := 0


	if isPresent {
		downloader.FileMode = 1
		for fileIndex, v := range metaInfo.BMaps["info"].BLists["files"].BMaps {
			filePath := v.BLists["path"].Strings[0]
			fileLength, _ := strconv.Atoi(v.Strings["length"])
			totalLength += fileLength
			newFile := createFileProperties(fileProperties, filePath, fileLength, fileIndex)
			fileProperties = append(fileProperties, newFile)
		}
	} else {
		downloader.FileMode = 0
		filePath := metaInfo.BMaps["info"].Strings["name"]
		fileLength, _ := strconv.Atoi(metaInfo.BMaps["info"].Strings["length"])
		totalLength += fileLength
		fileProperties = append(fileProperties, createFileProperties(fileProperties, filePath, fileLength, 0))
	}
	downloader.subPieceLength = SubPieceLen
	initFile(downloader, fileProperties, totalLength, totalLength, nil, StoppedState)

	downloader.PieceSelectionMode = sequentialSelection
	downloader.SelectNewPiece = true

	fmt.Printf("\n downloader pieceLength %v , pieceLen %v, subPieceLength %v, n SubPieces %v \n", downloader.FileLength, downloader.pieceLength, downloader.subPieceLength, downloader.nSubPiece)
	return downloader
}

// resume a saved torrentFile
func resumeDownloader(path string) *TorrentDownloader {

	torrentDataJSON, errr := ioutil.ReadFile(path)

	if errr != nil {
		log.Fatal(errr)
	}

	println(torrentDataJSON)

	torrentData := new(SavedTorrentData)
	_ = json.Unmarshal(torrentDataJSON, torrentData)

	downloader := new(TorrentDownloader)

	downloader.Announce = torrentData.Announce
	downloader.AnnounceList = make([]string, 0)
	downloader.CreationDate = torrentData.CreationDate
	downloader.CreatedBy = torrentData.CreatedBy

	downloader.pieceLength = torrentData.PieceLength
	downloader.left = torrentData.Left
	downloader.Name = torrentData.Name
	downloader.subPieceLength = torrentData.SubPieceLen
	pieceHashPath := utils.GetPath(utils.PieceHashPath, downloader.Name, "piecesHash.sha1")
	pieceHash, _ := ioutil.ReadFile(pieceHashPath)
	downloader.piecesSha1Hash = string(pieceHash)
	downloader.Bitfield = make([]byte, len(torrentData.Bitfield))

	for i, b := range torrentData.Bitfield {
		downloader.Bitfield[i] = byte(b)
	}

	fmt.Printf("Bitfield \n %v\n", downloader.Bitfield)

	/*for _ , b := range torrentData.InfoHash{
	//	downloader.InfoHash += string(byte(b))
	}
	*/
	b, _ := hex.DecodeString(torrentData.InfoHashHex)
	downloader.InfoHash = string(b)
	downloader.InfoHashHex = torrentData.InfoHashHex

	//will create the pieces
	initFile(downloader, torrentData.FileProperties, torrentData.FileLen, torrentData.Left, downloader.Bitfield, torrentData.State)
	fmt.Printf("\ndownloader info\n downloader name : %v \n fileLen  %v \n pieceLen %v\n subPieceLength %v\n infoHash %v\n", downloader.Name, downloader.FileLength, downloader.pieceLength, downloader.subPieceLength, downloader.InfoHash)
	downloader.PieceSelectionMode = sequentialSelection
	downloader.SelectNewPiece = true
	return downloader
}

func (downloader *TorrentDownloader) SetState(state int) {
	atomic.StoreInt32(downloader.State, int32(state))
}

// saves the current state of a download, so it can be resumed later
func (downloader *TorrentDownloader) saveTorrent() {

	SavedTorrentData := new(SavedTorrentData)
	SavedTorrentData.Announce = downloader.Announce
	SavedTorrentData.InfoHash = downloader.InfoHash
	SavedTorrentData.InfoHashHex = downloader.InfoHashHex
	SavedTorrentData.Comment = downloader.Comment
	SavedTorrentData.CreatedBy = downloader.CreatedBy
	SavedTorrentData.CreationDate = downloader.CreationDate
	SavedTorrentData.PieceLength = downloader.pieceLength
	///	SavedTorrentData.PiecesSha1 = downloader.piecesSha1Hash

	SavedTorrentData.Left = downloader.left
	SavedTorrentData.FileLen = downloader.FileLength
	SavedTorrentData.SubPieceLen = downloader.subPieceLength
	SavedTorrentData.Name = downloader.Name
	SavedTorrentData.State = int(atomic.LoadInt32(downloader.State))
	SavedTorrentData.Bitfield = make([]int, len(downloader.Bitfield))
	SavedTorrentData.InfoHashByte = make([]int, len(downloader.infoHashByte))

	for i, _ := range downloader.Bitfield {
		SavedTorrentData.Bitfield[i] = int(downloader.Bitfield[i])
	}

	for i, _ := range downloader.infoHashByte {
		SavedTorrentData.InfoHashByte[i] = int(downloader.infoHashByte[i])
	}

	SavedTorrentData.FileProperties = downloader.FileProperties

	SavedTorrentDataBuffer := bytes.NewBuffer(make([]byte, 0))
	SavedTorrentDataJSONEncoder := json.NewEncoder(SavedTorrentDataBuffer)
	SavedTorrentDataJSONEncoder.SetEscapeHTML(false)
	SavedTorrentDataJSONEncoder.SetIndent("", "")
	_ = SavedTorrentDataJSONEncoder.Encode(SavedTorrentData)

	pieceHashPath := utils.GetPath(utils.PieceHashPath, downloader.Name, "piecesHash.sha1")

	if _, err := os.Stat(pieceHashPath); os.IsNotExist(err) {
		_ = ioutil.WriteFile(pieceHashPath, []byte(downloader.piecesSha1Hash), os.ModePerm)
	}

	savedTorrentDataPath := utils.GetPath(utils.TorrentDataPath, downloader.Name, downloader.Name+".json")
	wErr := ioutil.WriteFile(savedTorrentDataPath, SavedTorrentDataBuffer.Bytes(), os.ModePerm)
	if wErr != nil {
		fmt.Printf("%v", wErr)
		//os.Exit(222)
	}

}
func (downloader *TorrentDownloader) saveTorrentForUiUpdate() {
	SavedTorrentData := new(SavedTorrentData)
	SavedTorrentData.Announce = downloader.Announce
	SavedTorrentData.InfoHash = downloader.InfoHash
	SavedTorrentData.Comment = downloader.Comment
	SavedTorrentData.CreatedBy = downloader.CreatedBy
	SavedTorrentData.CreationDate = downloader.CreationDate
	SavedTorrentData.PieceLength = downloader.pieceLength
	///	SavedTorrentData.PiecesSha1 = downloader.piecesSha1Hash

	i := []byte(downloader.piecesSha1Hash)

	fmt.Printf("len sha1 %v", i)
	SavedTorrentData.Left = downloader.left
	SavedTorrentData.FileLen = downloader.FileLength
	SavedTorrentData.SubPieceLen = downloader.subPieceLength
	SavedTorrentData.Name = downloader.Name
	SavedTorrentData.State = int(atomic.LoadInt32(downloader.State))

	pieces := make([]*Piece, downloader.nPiece)

	for i, p := range downloader.Pieces {
		newPiece := new(Piece)

		downloader.PiecesMutex.Lock()
		newPiece.State = p.State
		newPiece.PieceIndex = p.PieceIndex
		downloader.PiecesMutex.Unlock()
		pieces[i] = newPiece
	}

	SavedTorrentData.FileProperties = downloader.FileProperties

	SavedTorrentData.Pieces = pieces

	SavedTorrentDataBuffer := bytes.NewBuffer(make([]byte, 0))
	SavedTorrentDataJSONEncoder := json.NewEncoder(SavedTorrentDataBuffer)
	SavedTorrentDataJSONEncoder.SetEscapeHTML(false)
	SavedTorrentDataJSONEncoder.SetIndent("", "")
	_ = SavedTorrentDataJSONEncoder.Encode(SavedTorrentData)
}

// initialize pieces of file
func initFile(downloader *TorrentDownloader, fileProperties []*fileProperty, fileTotalLength, left int, bitfield []byte, torrentStatus int) {
	downloader.State = new(int32)
	atomic.StoreInt32(downloader.State, int32(torrentStatus))

	for _, f := range fileProperties {
		downloader.FileProperties = append(downloader.FileProperties, f)
	}

	downloader.FileLength = fileTotalLength
	downloader.left = left
	downloader.subPieceLength = int(math.Min(float64(downloader.pieceLength), float64(downloader.subPieceLength)))
	downloader.nSubPiece = int(math.Ceil(float64(downloader.pieceLength) / float64(downloader.subPieceLength)))
	downloader.nPiece = int(math.Ceil(float64(downloader.FileLength) / float64(downloader.pieceLength)))
	fmt.Printf("nPieces %v\n", downloader.nPiece)

	downloader.Pieces = make([]*Piece, downloader.nPiece)
	downloader.PieceAvailability = arraylist.New()
	downloader.pieceAvailabilityMutex = new(sync.RWMutex)
	downloader.PieceSorter = new(PieceSorter)
	downloader.PieceSorter.neededPieces = make([]*Piece, 0)
	downloader.Bitfield = make([]byte, int(math.Ceil(float64(downloader.nPiece)/float64(8))))
	for i := range downloader.Pieces {

		pieceLen := downloader.pieceLength
		if i == downloader.nPiece-1 {
			if downloader.FileLength%downloader.pieceLength != 0 {
				pieceLen = downloader.FileLength % downloader.pieceLength
				println("pieceLen")
				println(pieceLen)
				println(downloader.FileLength)

			}
		}

		thisPieceStatus := EmptyPiece

		if bitfield != nil {
			currentPieceBitFieldIndex := int(math.Ceil(float64(i / 8)))
			currentPieceBitFieldBitIndex := i % 8
			fmt.Printf("bitfiledIndex %v, bitIndex %v\n", currentPieceBitFieldIndex, currentPieceBitFieldBitIndex)

			if utils.IsBitOn(bitfield[currentPieceBitFieldIndex], currentPieceBitFieldBitIndex) {
				fmt.Printf("bit is on\n")
				thisPieceStatus = CompletePiece
			}

		}
		downloader.Pieces[i] = NewPiece(downloader, i, pieceLen, thisPieceStatus)
		//downloader.Pieces[i].Availability = i

		if thisPieceStatus != CompletePiece {

			downloader.PieceSorter.neededPieces = append(downloader.PieceSorter.neededPieces, downloader.Pieces[i])
		}

		downloader.PieceAvailability.Add(downloader.Pieces[i])
	}

	pieceIndex := 0

	// processes overlapping pieces
	// determine which subPiece belongs to a file and the position where it starts
	for i := 0; i < len(downloader.FileProperties); i++ {
		f := downloader.FileProperties[i]
		pos := new(pos)
		pos.fileIndex = i
		for pos.end != f.EndIndex {
			currentPiece := downloader.Pieces[pieceIndex]
			pos.start = int(math.Max(float64(currentPiece.pieceStartIndex), float64(f.StartIndex)))
			pos.end = int(math.Min(float64(currentPiece.pieceEndIndex), float64(f.EndIndex)))
			pos.writingIndex = pos.start - f.StartIndex
			currentPiece.position = append(currentPiece.position, *pos)
			if currentPiece.pieceEndIndex == pos.end {
				pieceIndex++
			}

		}

	}

	downloader.downloadRateMutex = new(sync.Mutex)
	downloader.addPieceChannel = make(chan *MSG, 5)
	go downloader.fileAssembler()
}

// store the path , len , start and end index of file
func createFileProperties(fileProperties []*fileProperty, filePath string, fileLength int, fileIndex int) *fileProperty {
	f := new(fileProperty)

	f.Path = filePath
	f.Length = fileLength

	if fileIndex == 0 {
		f.StartIndex = 0
	} else {
		f.StartIndex = fileProperties[fileIndex-1].EndIndex
	}

	f.EndIndex = f.StartIndex + f.Length

	return f
}

// calculates the info hash based on pieces provided in the .torrent file
func GetInfoHash(dict *parser.BMap) (string, [20]byte) {

	// InnerStartingPosition leaves out the key

	startingPosition := dict.BMaps["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.BMaps["info"].KeyInfo.EndingPosition

	torrentFileString := parser.ToBencode(dict)

	torrentFileByte := []byte(torrentFileString)

	infoBytes := torrentFileByte[startingPosition:endingPosition]

	infoHash := sha1.Sum(infoBytes)
	infoHashSlice := infoHash[:]
	println(hex.EncodeToString(infoHashSlice))

	return string(infoHashSlice), infoHash

}

func (downloader *TorrentDownloader) isPieceValid(piece *Piece) bool {

	validHash := downloader.piecesSha1Hash[piece.PieceIndex*20 : (piece.PieceIndex*20)+20]

	fmt.Printf("piece Index %v\n", piece.PieceIndex)

	pieceHash := sha1.Sum(downloader.PieceBuffer)
	pieceHashSlice := pieceHash[:]

	fmt.Printf("\n pieceHash :\n")
	fmt.Printf(" %v\n", string(pieceHashSlice))
	fmt.Printf("\n# neededSubpiece : %v\n", len(piece.neededSubPiece))
	fmt.Printf("\n# PieceCounter : %v\n", downloader.torrent.PieceCounter)
	fmt.Printf("\n currentValidHash :\n")
	fmt.Printf(" %v\n", validHash)
	fmt.Printf("Piece are equal : %v\n", validHash == string(pieceHashSlice))

	return validHash == string(pieceHashSlice)
}

//	Creates a Piece object and initialize subPieceRequest for this piece
func NewPiece(downloader *TorrentDownloader, PieceIndex, pieceLength int, status int) *Piece {
	newPiece := new(Piece)

	newPiece.Len = pieceLength
	newPiece.owners = make(map[string]*Peer)
	newPiece.Availability = 0
	newPiece.PieceIndex = PieceIndex
	newPiece.pieceStartIndex = PieceIndex * downloader.pieceLength
	newPiece.pieceEndIndex = newPiece.pieceStartIndex + newPiece.Len
	newPiece.nSubPiece = int(math.Ceil(float64(pieceLength) / float64(downloader.subPieceLength)))
	//newPiece.Pieces = make([]byte, newPiece.pieceLength)
	newPiece.State = status
	newPiece.subPieceMask = make([]byte, int(math.Ceil(float64(newPiece.nSubPiece)/float64(8))))
	newPiece.pendingRequestMutex = new(sync.RWMutex)
	newPiece.pendingRequest = make([]*PieceRequest, 0)

	newPiece.neededSubPiece = make([]*PieceRequest, newPiece.nSubPiece)
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

		pendingRequest := new(PieceRequest)

		pendingRequest.startIndex = i * SubPieceLen

		msg := MSG{ID: RequestMsg, PieceIndex: newPiece.PieceIndex, BeginIndex: pendingRequest.startIndex, PieceLen: newPiece.SubPieceLen}

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
func (downloader *TorrentDownloader) SortPieceByAvailability() {
	downloader.pieceAvailabilityMutex.Lock()
	if time.Now().Sub(downloader.PieceAvailabilityTimeStamp) >= time.Second*5 {
		downloader.PieceAvailability.Sort(pieceAvailabilityComparator)
		downloader.PieceAvailabilityTimeStamp = time.Now()
	}
	downloader.pieceAvailabilityMutex.Unlock()
}

//Sort the needed pieces by their availability
type PieceSorter struct {
	neededPieces []*Piece
}

func (pieceSorter PieceSorter) Len() int {
	return len(pieceSorter.neededPieces)
}

func (pieceSorter PieceSorter) Less(i, j int) bool {
	return pieceSorter.neededPieces[i].Availability < pieceSorter.neededPieces[j].Availability
}

func (pieceSorter PieceSorter) Swap(i, j int) {
	pieceSorter.neededPieces[i], pieceSorter.neededPieces[j] = pieceSorter.neededPieces[j], pieceSorter.neededPieces[i]
}

func (downloader *TorrentDownloader) fileAssembler() {

	for {
		msg := <-downloader.addPieceChannel
		println("receiving piece .........")
		if atomic.LoadInt32(downloader.State) == int32(StartedState) {
			currentPiece := downloader.Pieces[msg.PieceIndex]
			subPieceIndex := int(math.Ceil(float64(msg.BeginIndex / SubPieceLen)))
			subPieceBitMaskIndex := int(math.Ceil(float64(subPieceIndex / 8)))
			subPieceBitIndex := subPieceIndex % 8
			downloader.torrent.PieceCounter++

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
				msg.Peer.mutex.Unlock()
			}
			isEmpty := utils.IsBitOn(currentPiece.subPieceMask[subPieceBitMaskIndex], subPieceBitIndex) == false
			currentPiece.subPieceMask[subPieceBitMaskIndex] = utils.BitMask(currentPiece.subPieceMask[subPieceBitMaskIndex], 1, []int{subPieceBitIndex}, 1)

			// if we haven't already receive this piece , we saved it
			if isEmpty {
				if msg.PieceIndex != downloader.CurrentPieceIndex {
					fmt.Printf("msg Piece App %v current Piece App %v", msg.PieceIndex, downloader.CurrentPieceIndex)
					os.Exit(223444)
				}

				currentPiece.pendingRequestMutex.Lock()
				for i := len(currentPiece.pendingRequest) - 1; i >= 0; i-- {
					if msg.BeginIndex == currentPiece.pendingRequest[i].startIndex {

						currentPiece.completedRequest = append(currentPiece.completedRequest, currentPiece.pendingRequest[i])

						currentPiece.pendingRequest[i], currentPiece.pendingRequest[len(currentPiece.pendingRequest)-1] = currentPiece.pendingRequest[len(currentPiece.pendingRequest)-1], nil

						currentPiece.pendingRequest = currentPiece.pendingRequest[:len(currentPiece.pendingRequest)-1]

						break
					}
				}
				currentPiece.pendingRequestMutex.Unlock()

				///fmt.Printf("Piece Counter %v\n",downloader.torrent.PieceCounter)
				currentPiece.CurrentLen += len(msg.Piece)
				downloader.TotalDownloaded += msg.PieceLen
				downloader.left -= msg.PieceLen
				///fmt.Printf("downloaded %v mb in %v \n", downloader.TotalDownloaded/1000000.0, time.Now().Sub(downloader.timeS))

				copy(downloader.PieceBuffer[msg.BeginIndex:msg.BeginIndex+msg.PieceLen], msg.Piece)

				//fmt.Printf("begin Index : %v, end index : %v \n",msg.BeginIndex,msg.BeginIndex+msg.PieceLen)

				// piece is complete add it to the downloader
				isPieceComplete := currentPiece.Len == currentPiece.CurrentLen
				fmt.Printf("currentPiece request # %v, completed request # %v, pending request # %v\n", len(currentPiece.neededSubPiece), len(currentPiece.completedRequest), len(currentPiece.pendingRequest))

				// once a piece is complete we write the it to downloader
				if isPieceComplete {
					var errWritingPiece error
					fmt.Printf(" currentPiece.pieceLength %v, currentPiece.downloaded %v\n", currentPiece.Len, currentPiece.CurrentLen)

					if !downloader.isPieceValid(currentPiece) {

						fmt.Printf("currentPiece request # %v, completed request # %v, pending request # %v\n", len(currentPiece.neededSubPiece), len(currentPiece.completedRequest), len(currentPiece.pendingRequest))
						fmt.Printf("currentLen %v, left %v, totalDownloaded %v\n", currentPiece.CurrentLen, downloader.left, downloader.TotalDownloaded)
						downloader.TotalDownloaded -= currentPiece.Len
						downloader.left += currentPiece.Len

						*currentPiece = *NewPiece(downloader, currentPiece.PieceIndex, currentPiece.Len, EmptyPiece)

						println()
						fmt.Printf("currentPiece request # %v, completed request # %v, pending request # %v\n", len(currentPiece.neededSubPiece), len(currentPiece.completedRequest), len(currentPiece.pendingRequest))
						fmt.Printf("currentLen %v, left %v, totalDownloaded %v\n", currentPiece.CurrentLen, downloader.left, downloader.TotalDownloaded)
					} else {
						//downloader.SelectNewPiece = true
						//	fmt.Printf("piece %v is complete\n",currentPiece.PieceIndex)

						//	a piece can contains data that goes into multiple files
						// pos struct contains start and end index , that indicates into which downloader a particular slice of data goes
						for _, pos := range currentPiece.position {
							currentFile := downloader.FileProperties[pos.fileIndex]

							if _, err := os.Stat(utils.TorrentHomeDir); os.IsNotExist(err) {
								_ = os.MkdirAll(utils.TorrentHomeDir, os.ModePerm)
							}

							var f *os.File
							var errOpenFile error

							var path string
							if len(downloader.FileProperties) > 1 {
								path = utils.GetPath(utils.DownloadedFile, downloader.Name, currentFile.Path)
							} else {
								path = utils.GetPath(utils.DownloadedFile, "", currentFile.Path)
							}

							f, errOpenFile = os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.ModePerm)
							if errOpenFile != nil {
								println(path)
								log.Fatal("could not open File")
							}

							//getting the length of the piece
							pieceRelativeLen := pos.end - pos.start
							pieceRelativeStart := pos.start - (currentPiece.PieceIndex * downloader.pieceLength)
							pieceRelativeEnd := pieceRelativeStart + pieceRelativeLen

							_, errWritingPiece = f.WriteAt(downloader.PieceBuffer[pieceRelativeStart:pieceRelativeEnd], int64(pos.writingIndex))
							if errWritingPiece != nil {
								log.Fatal(errWritingPiece)
							}

							_ = f.Close()
						}
						currentPiece.State = CompletePiece

						for i, p := range downloader.PieceSorter.neededPieces {
							if p.PieceIndex == currentPiece.PieceIndex {

								downloader.PieceSorter.neededPieces[i], downloader.PieceSorter.neededPieces[downloader.PieceSorter.Len()-1] = downloader.PieceSorter.neededPieces[downloader.PieceSorter.Len()-1], nil
								downloader.PieceSorter.neededPieces = downloader.PieceSorter.neededPieces[:downloader.PieceSorter.Len()-1]
								break
							}
						}

						currentPieceBitFieldIndex := int(math.Ceil(float64(currentPiece.PieceIndex / 8)))
						currentPieceBitFieldBitIndex := currentPiece.PieceIndex % 8
						downloader.Bitfield[currentPieceBitFieldIndex] = utils.BitMask(downloader.Bitfield[currentPieceBitFieldIndex], 1, []int{currentPieceBitFieldBitIndex}, 1)
						fmt.Printf("%v", downloader.Bitfield)

						downloader.saveTorrent()
						///currentPiece.Pieces = make([]byte, 0)
						downloader.PieceBuffer = make([]byte, 0)
						downloader.SelectNewPiece = true
					}

				} else {
					//	fmt.Printf("not complete yet , %v/%v pieceindex : %v\n",currentPiece.pieceLength,currentPiece.downloaded,currentPiece.PieceIndex)
					currentPiece.State = InProgressPiece
					downloader.SelectNewPiece = false
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

			downloader.downloadRateMutex.Lock()
			downloader.tempDownloadCounter += len(msg.Piece)
			if time.Now().Sub(downloader.downloadRateTimeStamp) > time.Second*3 {
				duration := time.Now().Sub(downloader.downloadRateTimeStamp)
				downloader.DownloadRate = float64(downloader.tempDownloadCounter) / duration.Seconds()
				downloader.downloadRateTimeStamp = time.Now()
				downloader.tempDownloadCounter = 0
			}
			downloader.downloadRateMutex.Unlock()

		}

	}

	/// aggregate stats about a peer
	// will be used to determine which peer to unchoke
	// and which that we will prefer when requesting subPieces

	//	fmt.Printf("peer: %v , download rate: %v byte/s \n", msg.Peer.ip, peer.DownloadRate)

}

//	Selects Pieces that need to be downloaded
//	When a piece is completely downloaded , a new one is selected
func (downloader *TorrentDownloader) PieceRequestManager(periodic *periodicFunc) {
	println("request piece !!!!!!!!!!!")
	if atomic.LoadInt32(downloader.State) == StartedState {

		downloader.PiecesMutex.Lock()

		if downloader.SelectNewPiece {
			var selectedPiece *Piece
			if downloader.nPiece-len(downloader.PieceSorter.neededPieces) <= 5 {
				downloader.PieceSelectionMode = randomSelection
			} else {
				downloader.PieceSelectionMode = prioritySelection
			}

			if downloader.PieceSelectionMode == randomSelection {
				//fmt.Printf("switching piece # %v\n", downloader.CurrentPieceIndex)
				downloader.SelectNewPiece = false
				downloader.torrent.PieceCounter = 0
				randSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
				randN := randSeed.Intn(len(downloader.PieceSorter.neededPieces))
				selectedPiece = downloader.PieceSorter.neededPieces[randN]
				downloader.CurrentPieceIndex = selectedPiece.PieceIndex

			} else {
				sort.Sort(downloader.PieceSorter)
				selectedPiece = downloader.PieceSorter.neededPieces[0]
				downloader.CurrentPieceIndex = selectedPiece.PieceIndex

			}
			if selectedPiece.Len == 0 {
				os.Exit(23)
			}
			downloader.PieceBuffer = make([]byte, selectedPiece.Len)
			downloader.SelectNewPiece = false
		} else {

			//fmt.Printf("not complete yet , currentLen %v , actual pieceLength %v\n",currentPiece.downloaded,currentPiece.pieceLength)
		}
		currentPiece := downloader.Pieces[downloader.CurrentPieceIndex]
		if !downloader.SelectNewPiece && len(currentPiece.neededSubPiece) == 0 && len(currentPiece.pendingRequest) == 0 {
			log.Printf("I messed up")
		}
		currentPiece.pendingRequestMutex.Lock()
		nSubPieceRequest := len(currentPiece.pendingRequest)

		//	the number of subPiece request shall not exceed maxRequest
		//	new requests are sent if the previous request have been fulfilled
		if nSubPieceRequest < maxRequest {
			nReq := maxRequest - len(currentPiece.pendingRequest)

			nReq = int(math.Min(float64(nReq), float64(len(currentPiece.neededSubPiece))))

			i := 0
			randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))

			//fmt.Printf("n pending Request : %v  nReq: %v needed Subp : %v\n",len(currentPiece.pendingRequest),nReq,len(currentPiece.neededSubPiece))

			for i < nReq {
				// 	subPieces request are randomly selected
				randomN := randomSeed.Intn(len(currentPiece.neededSubPiece))

				req := currentPiece.neededSubPiece[randomN]

				//fmt.Printf("sending request\n")
				err := downloader.requestPiece(req)

				// once a request is added to the job JobQueue, it is removed from the needed subPiece / prevents from being reselected
				if err == nil {
					currentPiece.neededSubPiece[randomN] = currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1]
					currentPiece.neededSubPiece[len(currentPiece.neededSubPiece)-1] = nil
					currentPiece.neededSubPiece = currentPiece.neededSubPiece[:len(currentPiece.neededSubPiece)-1]
				} else {

					fmt.Printf("ERR : %v\n", err)
					//os.Exit(99)
				}
				i++
				//	fmt.Printf("sending new Request.....\n")

			}
			//fmt.Printf("sending new Request2.....\n")

		}

		// re-sends request that have been pending for a certain amounts of time
		nPendingRequest := len(currentPiece.pendingRequest)

		for r := 0; r < nPendingRequest; r++ {
			pendingRequest := currentPiece.pendingRequest[r]

			if time.Now().Sub(pendingRequest.timeStamp) >= maxWaitingTime {
				//fmt.Printf("sending request\n")
				_ = downloader.requestPiece(pendingRequest)
			}

			//fmt.Printf("Resending Request.....\n")

		}
		currentPiece.pendingRequestMutex.Unlock()

		downloader.PiecesMutex.Unlock()

		//	fmt.Printf("Resending Request 3.....\n")
	}

}

//	Adds a request for piece to the a job Queue
//	Params :
//	subPieceRequest *PieceRequest : contains the raw request msg
func (downloader *TorrentDownloader) requestPiece(subPieceRequest *PieceRequest) error {

	//println("im not stuck! 1")
	downloader.torrent.PeerSwarm.SortPeerByDownloadRate()
	var err error

	peerOperation := new(peerOperation)
	peerOperation.operation = isPeerFree
	peerOperation.freePeerChannel = make(chan *Peer)
	currentPiece := downloader.Pieces[downloader.CurrentPieceIndex]
	var peer *Peer
	/*
		peerIndex := 0

		isPeerFree := false

		for !isPeerFree && peerIndex < downloader.PeerSwarm.PeerByDownloadRate.Size(){
			fmt.Printf("looking for a suitable peer | # of available peer : %v\n", downloader.PeerSwarm.PeerByDownloadRate.Size())
			//	fmt.Printf("\nchoking peer # %v\n",downloader.chokeCounter)
			fmt.Printf("looking for a suitable peer | # of available peer active connection: %v\n", downloader.PeerSwarm.activeConnection.Size())
			peerI, found := downloader.PeerSwarm.PeerByDownloadRate.Get(peerIndex)

			if found && peerI != nil {
				peer = peerI.(*Peer)
				peerIndex = peerIndex % downloader.PeerSwarm.PeerByDownloadRate.Size()

				if peer != nil {
					isPeerFree = peer.isPeerFree()

					//	fmt.Printf("peer is not free, # pendingg Req : %v | peer App : %v\n ",len(peer.peerPendingRequest),peerIndex)
				} else {
					//fmt.Printf("peer is nil ")

					isPeerFree = false
				}
			} else {
			}

		}
	*/
	downloader.torrent.PeerSwarm.peerOperation <- peerOperation
	peer = <-peerOperation.freePeerChannel

	if peer != nil {
		subPieceRequest.msg.Peer = peer
		downloader.torrent.jobQueue.AddJob(subPieceRequest.msg)
		peer.mutex.Lock()
		peer.peerPendingRequest = append(peer.peerPendingRequest, subPieceRequest)
		peer.mutex.Unlock()

		subPieceRequest.timeStamp = time.Now()

		if subPieceRequest.status == nonStartedRequest {
			subPieceRequest.status = pendingRequest
			currentPiece.pendingRequest = append(currentPiece.pendingRequest, subPieceRequest)
		}
		//fmt.Printf("request id %v %v \n",subPieceRequest.startIndex, subPieceRequest.msg)

	} else {
		err = errors.New("request is not sent ")
	}

	return err
}
