package PeerProtocol

import (
	"time"
)

const (
	startedState   = 0
	stoppedState   = 1
	completedState = 2
	SubPieceLen    = 16384
	maxWaitingTime = time.Millisecond*120
	maxRequest	= 5
	peerDownloadRatePeriod = time.Second*10
)

