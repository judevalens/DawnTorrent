package tracker

import (
	"DawnTorrent/interfaces"
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"time"
)

const (
	httpScheme = "https"
	udpScheme  = "udp"
)

type Announcer struct {
	state          trackerRequestState
	infoHash       string
	mainTrackerUrl *url.URL
	trackerUrls    []*url.URL
	interval       time.Duration
	timer          *time.Timer
	torrentManager interfaces.TorrentManagerI
	peerManager    interfaces.PeerManagerI
	strategy       scrapeStrategy
}

func NewAnnouncer(announcerUrlString, infoHash string, manager interfaces.TorrentManagerI,peerManager interfaces.PeerManagerI) (*Announcer, error) {
	var baseTracker *Announcer
	trackerURL, err := url.Parse("udp://tracker.opentrackr.org:1337")

	if err != nil {
		return nil, err
	}
	fmt.Printf("info hash : %v\n", infoHash)
	baseTracker = &Announcer{
		infoHash:       infoHash,
		mainTrackerUrl: trackerURL,
		timer:          time.NewTimer(time.Nanosecond),
		torrentManager: manager,
		peerManager: peerManager,
		state:          &initialRequest{baseTracker},
	}


	for _, s := range manager.GetAnnounceList() {
		newUrl, err := url.Parse(s)

		if err != nil{
			log.Fatal(err)
		}

		baseTracker.trackerUrls = append(baseTracker.trackerUrls,newUrl)
	}

	baseTracker.setTrackerStrategy(trackerURL)

	log.Printf("url: %v, url scheme : %v", announcerUrlString, trackerURL.Scheme)


	log.Printf("creating new Announcer: %v", reflect.TypeOf(baseTracker))

	return baseTracker, nil
}

func (t *Announcer) setTrackerStrategy(url *url.URL){
	t.state.cancel()
	if url.Scheme == "https" || url.Scheme == "http" {
		t.strategy = &httpTracker2{
			t,
		}
	} else {
		t.strategy = &udpTracker2{
			t,
		}
	}
}

func (t *Announcer) getCurrentState() string {
	switch t.torrentManager.GetState() {
	case interfaces.StartTorrent:
		return "started"
	case interfaces.StopTorrent:
		return "stopped"
	case interfaces.CompleteTorrent:
		return "completed"
	default:
		return ""
		
	}
}

func (t *Announcer) getCurrentStateInt() int  {
	return t.torrentManager.GetState()
}


func (t *Announcer) getTransferStats() (int, int, int) {
	return t.torrentManager.GetStats()
}

func (t *Announcer) StartScrapper(ctx context.Context) {

	t.state = &initialRequest{scrapper: t}
	t.state.handle()
}

func (t *Announcer) stopTracker() {
	t.state.cancel()
}
func (t *Announcer) resetTracker(duration time.Duration) {
	t.state.reset(duration)
}

type trackerRequestState interface {
	handle()
	cancel()
	reset(duration time.Duration)
}

type initialRequest struct {
	scrapper *Announcer
}

func (i *initialRequest) handle() {
	var err error
	var interval int
	log.Printf("trackerUrl: %v", i.scrapper.mainTrackerUrl.String())
	log.Printf("sending initial 2 Announcer request, trackerType : %v", reflect.TypeOf(i.scrapper))
	interval, err = i.scrapper.strategy.handleRequest()


	if errors.Is(err,os.ErrDeadlineExceeded){
		interval, err =  loopThroughList(i.scrapper)
	}

	if err != nil {
		log.Fatalf("err:  %v", err)
		return
	}

	print("interval")
	print(interval)

	i.scrapper.interval, _ = time.ParseDuration(strconv.Itoa(interval)+"s")
	i.scrapper.state = &recurringRequest{scrapper: i.scrapper}
	log.Print("launching Announcer request")
	i.scrapper.state.handle()

}


/*
	iterates through a torrent tracker url list to find a responsive server
 */
func loopThroughList(scrapper *Announcer)(int,error){
	var err error
	var interval int
	for _, trackerUrl := range scrapper.trackerUrls {
		scrapper.mainTrackerUrl = trackerUrl
		scrapper.setTrackerStrategy(trackerUrl)
		interval, err = scrapper.strategy.handleRequest()
		if err != nil {
			log.Printf("loopThroughList err : ^%v",err)
			continue
		}

		return interval,err
	}
	//TODO should probably aggregate all the known errors in some file
	return -1,errors.New("failed to find a tracker")
}

func (i *initialRequest) cancel() {
}

func (i *initialRequest) reset(time.Duration) {
}

type recurringRequest struct {
	scrapper *Announcer
}

func (r *recurringRequest) handle() {
	log.Printf("next tracker request will fire in %v", r.scrapper.interval)
	r.scrapper.timer = time.AfterFunc(r.scrapper.interval, func() {
		log.Printf("sending initial Announcer request, trackerType : %v", reflect.TypeOf(r.scrapper))

		interval, err := r.scrapper.strategy.handleRequest()
		if err != nil {
			if errors.Is(err,os.ErrDeadlineExceeded){
				interval, err =  loopThroughList(r.scrapper)
			}
			return
		}

		r.scrapper.interval, _ = time.ParseDuration(strconv.Itoa(interval)+"s")
		r.scrapper.state = &recurringRequest{scrapper: r.scrapper}
		r.handle()
	})

}

func (r *recurringRequest) cancel() {
	r.scrapper.timer.Stop()
	r.scrapper.state = &initialRequest{}
}

func (r *recurringRequest) reset(duration time.Duration) {
	r.scrapper.timer.Stop()
	r.scrapper.timer.Reset(duration)
}
