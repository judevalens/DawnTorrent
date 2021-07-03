package state

import (
	"DawnTorrent/interfaces"
)

type Subscriber interface {
	UpdateState(Manager interfaces.TorrentManagerI)
}
