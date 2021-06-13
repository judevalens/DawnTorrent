package app

import (
	"time"
)

type BlockRequest struct {
	RequestMsg
	Id         string
	FullFilled bool
	TimeStamp  time.Time
	Providers  []*Peer
}
