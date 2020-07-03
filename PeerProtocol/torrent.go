package PeerProtocol

import (
	"github.com/emirpasic/gods/lists/arraylist"
	"sync"
	"time"
)

const (
	EmptyPiece      = 0
	CompletePiece   = 0
	InProgressPiece = 0
)

type TorrentFile struct {
	Announce       string
	AnnounceList   []string
	Comment        string
	CreatedBy      string
	CreationDate   string
	Encoding       string
	piecesSha1Hash string
	FileMode       int
	filesMutex     sync.RWMutex
	Files          []*fileInfo
	PiecesMutex    sync.RWMutex
	Pieces         []*Piece
	nPiece         int
	FileLen        int
	subPieceLen    int
	nSubPiece      int
	InfoHash       string
	infoHashByte   [20]byte
	InfoHashHex 			string
	pieceLength    int
	Name           string
	NeededPiece                map[int]*Piece
	PieceHolder				[]*byte
	currentPiece               *Piece
	PieceSelectionBehavior     string
	SelectNewPiece             bool
	CurrentPieceIndex          int
	torrent                    *Torrent
	timeS                      time.Time
	TotalDownloaded            int
	left                       int
	uploaded                   int
	pieceAvailabilityMutex     *sync.RWMutex
	PieceAvailability          *arraylist.List
	PieceAvailabilityTimeStamp time.Time
	Status                     *int32
	downloadRateMutex          *sync.Mutex
	tempDownloadCounter        int
	DownloadRate               float64
	downloadRateTimeStamp      time.Time
}

type fileInfo struct {
	Path       string
	Length     int
	FileIndex  int
	StartIndex int
	EndIndex   int
}

type Piece struct {
	Len                 int
	CurrentLen          int
	SubPieceLen         int
	State               int
	Pieces              []byte
	PieceIndex          int
	Availability        int
	owners              map[string]*Peer
	subPieceMask        []byte
	pieceStartIndex     int
	pieceEndIndex       int
	position            []pos
	pendingRequestMutex *sync.RWMutex
	pendingRequest      []*PieceRequest
	neededSubPiece      []*PieceRequest
	nSubPiece           int
	AvailabilityIndex   int
}

type PieceRequest struct {
	peerID        string
	startIndex    int
	timeStamp     time.Time
	len           int
	msg           *MSG
	status        int
	subPieceIndex int
}


// a piece can contains data that goes into multiple files
// pos struct contains start and end index , that indicates into which file a particular slice of data goes
type pos struct {
	// id that represent a file
	fileIndex    int
	// index that indicates where to start cutting data within a piece
	start        int
	// index that indicates where to stop (not inclusive) cutting data within a piece
	end          int
	// index that indicates at which position to write the pieces within file
	writingIndex int
}

type SavedTorrentData struct {
	Announce               string
	AnnounceList           []string
	Comment                string
	CreatedBy              string
	CreationDate           string
	Encoding               string
	InfoHash               string
	InfoHashHex 			string
	piecesHash             string
	PiecesSha1             string
	Left                   int
	FileLen                int
	PieceLength            int
	SubPieceLen            int
	nPiece                 int
	Pieces                 []*Piece
	FileInfos              []*fileInfo
	Name                   string
	State                  int
	PieceSelectionBehavior int
}


type TorrentIPCData struct {
	Name         string
	Path         string
	InfoHash     string
	Len          int
	CurrentLen   int
	PiecesStatus []bool
	State        int
	Command      int
	AddMode      int
	FileInfos    []*fileInfo
	DownloadRate				float64

}



