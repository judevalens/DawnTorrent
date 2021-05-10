package protocol

import (
	"DawnTorrent/parser"
	"context"
	"net"
)

const (
	StartTorrent    = iota
	StopTorrent     = iota
	CompleteTorrent = iota
)

type TorrentManagerI interface {
	GetState() int
	GetStats()(int,int,int)
	GetAnnounceList() []string
}

type PeerManagerI interface {
	GetActivePeers() map[string]PeerI
	AddNewPeer(peers []PeerI)
	HandleMsgStream(ctx context.Context,peer PeerI) error
	CreatePeersFromUDPTracker(peersAddress []byte) []PeerI
	CreatePeersFromHTTPTracker(peersInfo []*parser.BMap) []PeerI
}

type PeerI interface {
	GetId()	string
	GetAddress() *net.TCPAddr
	SetConnection(conn *net.TCPConn)
	GetConnection() *net.TCPConn
	GetBitfield() []byte
	SetBitField(bitfield []byte)
}
