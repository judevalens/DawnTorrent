package ipc

const (
	AddTorrent = 0
	DeleteTorrent  = 1
	PauseTorrent = 2
	ResumeTorrent = 3
	CloseClient = 4
	InitClient = 5
	updateUI = 6
)

type torrentIPCData struct {
	command int
	name string
	path string
	len string
	currentLen string
	piecesStatus []bool
}


func CommandRouter (msg torrentIPCData){
	switch msg.command {
	case AddTorrent:
	case DeleteTorrent:
	case PauseTorrent:
	case ResumeTorrent:
	case InitClient:
	case CloseClient:

	}
}