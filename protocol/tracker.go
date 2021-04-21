package protocol

import (
	"DawnTorrent/parser"
	"DawnTorrent/utils"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type trackerRequestState interface {
	send()
	cancel()
	reset(duration time.Duration)
}

type initialRequest struct {
	manager TorrentManager
}

func (i initialRequest) send() {

	interval, err := i.manager.runTracker(10)
	if err != nil {
		return
	}
	i.manager.in
	i.manager.trackerState = recurringRequest{}
	i.manager.trackerState.send()
}

func (i initialRequest) cancel() {
	panic("implement me")
}

func (i initialRequest) reset(time.Duration) {

}

func (i initialRequest) handle() {

}

type recurringRequest struct {
	manager TorrentManager
	interval int
}

func (r recurringRequest) send() {

	r.manager.trackerTimer = time.AfterFunc(time.Duration(r.interval), func() {
		interval, err := r.manager.runTracker(10)
		if err != nil {
			return
		}

		r.manager.trackerTimer.Reset(time.Duration(interval))
	})
}

func (r recurringRequest) cancel() {
	panic("implement me")
}

func (r recurringRequest) reset(duration time.Duration) {
	panic("implement me")
}


type done struct {
	manager TorrentManager
}

func (d done) send() {
	d.manager.trackerState = initialRequest{}
	d.manager.trackerState.send()
}

func (d done) cancel() {
	if d.manager.trackerTimer == nil{
		log.Fatal("timer is nil")
	}
	d.manager.trackerTimer.Stop()
}

func (d done) reset(duration time.Duration) {
	d.manager.trackerTimer.Reset(duration)
}

type tracker struct {
	state trackerRequestState

	interval int
}

func (t tracker) starTracker()  {
	t.state =  initialRequest{}
	t.state.send()
}

func (t tracker) stopTracker()  {
	t.state.cancel()
}
func (t tracker) resetTracker(duration time.Duration)  {
	t.state.reset(duration)
}

type trackerRequestProtocol interface {
	send()
}


type httpTracker2 struct {
	infoHash string
}

func (trp httpTracker2) getCurrentState() string {
	switch 0 {
	case 0:
		return "started"
	case 1:
		return  "stopped"
	case 2:
		return  "completed"

	}
	return ""
}
func (trp httpTracker2) getAnnouncerUrl() string {
	switch 0 {
	case 0:
		return "started"
	case 1:
		return  "stopped"
	case 2:
		return  "completed"

	}
	return ""
}
func (trp httpTracker2) getTransferStats() (int,int,int) {

	return 0,0,0
}


func (trp httpTracker2) send() {
	myID := utils.MyID


	event := trp.getCurrentState()

	//fmt.Printf("%x\n", InfoHash)

	uploaded,downloaded,left := trp.getTransferStats()

	trackerRequestParam := url.Values{}
	trackerRequestParam.Add("info_hash", trp.infoHash)
	trackerRequestParam.Add("peer_id", myID)
	trackerRequestParam.Add("port", strconv.Itoa(utils.PORT))
	trackerRequestParam.Add("uploaded", strconv.Itoa(uploaded))
	trackerRequestParam.Add("downloaded", strconv.Itoa(downloaded))
	trackerRequestParam.Add("left", strconv.Itoa(left))
	trackerRequestParam.Add("event", event)

	trackerUrl := trp.getAnnouncerUrl() + "?" + trackerRequestParam.Encode()
	fmt.Printf("\n Param \n %v \n", trackerUrl)

	trackerResponseByte, _ := http.Get(trackerUrl)
	trackerResponse, _ := ioutil.ReadAll(trackerResponseByte.Body)

	fmt.Printf("%v\n", string(trackerResponse))

	trackerDictResponse := parser.UnmarshallFromArray(trackerResponse)

	requestErr, isPresent := trackerDictResponse.Strings["failure reason"]

	if isPresent {
		log.Fatalf("error: %v", errors.New(requestErr))
	}

	return nil, nil
}



type udpTracker2 struct {

}

func (trp udpTracker2) send() {

}
