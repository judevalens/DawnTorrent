package app

import (
	log "github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)

const (
	requestTimeOut    = time.Second * 5
	maxPendingRequest = 10
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
			log.Debugf("duration %v, has timedout %v",time.Now().Sub(request.TimeStamp),time.Now().Sub(request.TimeStamp) > requestTimeOut)

			if atomic.LoadInt64(&request.state) == fulfilled{
				return true
			}

			hasTimeOut := time.Now().Sub(request.TimeStamp) > requestTimeOut
			if hasTimeOut{
				 atomic.StoreInt64(&request.state,timeout)
			}
	return time.Now().Sub(request.TimeStamp) > requestTimeOut
}

func (request BlockRequest) hasNoProvider() bool  {
	log.Debugf("has no provider %v",len(request.Providers) == 0)

	for i, provider := range request.Providers {
		log.Debugf("%v- provider id %v",i, provider.id)
	}

	return len(request.Providers) == 0
}
