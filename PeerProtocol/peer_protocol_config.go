package PeerProtocol

import (
	"time"
)

const (
	StartedState           = 0
	StoppedState           = 1
	CompletedState         = 2
	SubPieceLen            = 16384
	maxWaitingTime         = time.Millisecond*120
	maxRequest             = 5
	peerDownloadRatePeriod = time.Second*10
)

