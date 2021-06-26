package app

import (
	"sync/atomic"
	"time"
)

const (
	requestTimeOut    = time.Millisecond * 10
	maxPendingRequest = 15
	fulfilled         = iota
	timeout           = iota
	unfulfilled       = iota
)
type BlockRequest struct {
	RequestMsg
	Id        string
	state     int64
	TimeStamp time.Time
	Providers []*Peer
}

func (request BlockRequest) timedOut() bool  {
			//log.Warning("duration %v, has timedout %v",time.Now().Sub(request.TimeStamp),time.Now().Sub(request.TimeStamp) > requestTimeOut)

			if atomic.LoadInt64(&request.state) == fulfilled{
				return false
			}

			hasTimeOut := time.Now().Sub(request.TimeStamp) > requestTimeOut
			if hasTimeOut{
				 atomic.StoreInt64(&request.state,timeout)
			}
	return time.Now().Sub(request.TimeStamp) > requestTimeOut
}

func (request BlockRequest) hasNoProvider() bool  {
	//log.Warning("has no provider %v",len(request.Providers) == 0)

	return len(request.Providers) == 0
}
