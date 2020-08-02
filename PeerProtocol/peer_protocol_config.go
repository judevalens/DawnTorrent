package PeerProtocol

import (
	"time"
)

const (
	StartedState           = 0
	StoppedState           = 1
	CompletedState         = 2
 sendTrackerRequest = 3

SubPieceLen            = 16384
	maxWaitingTime         = time.Millisecond*300
	maxRequest             = 8
	peerDownloadRatePeriod = time.Second*30
	sequentialSelection    = 0
	randomSelection = 1
	prioritySelection = 2

)

