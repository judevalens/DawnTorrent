package interfaces

import (
	"DawnTorrent/parser"
	"net"
	"sync"
)

const (
	StartTorrent    = iota
	StopTorrent     = iota
	CompleteTorrent = iota
)
type SyncOp func()

type TorrentManagerI interface {
	GetState() int
	GetStats() (int, int, int)
	GetAnnounceList() []string
	GetSyncChan() chan SyncOp
}

type PeerManagerI interface {
	GetActivePeers() map[string]PeerI
	AddNewPeer(peers ...PeerI)
	CreatePeersFromUDPTracker(peersAddress []byte) []PeerI
	CreatePeersFromHTTPTracker(peersInfo []*parser.BMap) []PeerI
}

type PeerI interface {
	GetId() string
	GetAddress() *net.TCPAddr
	SetConnection(conn *net.TCPConn)
	GetConnection() *net.TCPConn
	GetBitfield() []byte
	SetBitField(bitfield []byte)
	HasPiece(pieceIndex int) bool
	GetMutex() *sync.Mutex
	IsChoking() bool
	SetChoke(x bool)
	IsInterested() bool
	SetInterest(x bool)
	UpdateBitfield(pieceIndex int)
}
