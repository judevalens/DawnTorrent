package subsService

import (
	"DawnTorrent/rpc/torrent_state"
)

type Subscriber interface {
	UpdateState(state *torrent_state.TorrentState)
}
