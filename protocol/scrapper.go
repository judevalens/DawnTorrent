package protocol

import (
	"context"
	"log"
	"net/url"
	"reflect"
	"time"
)

const (
	httpScheme = "https"
	udpScheme  = "udp"
)

type Scrapper struct {
	state       trackerRequestState
	infoHash    string
	trackerURL  *url.URL
	interval    time.Duration
	timer       *time.Timer
	peerManager *peerManager
	strategy    scrapeStrategy
}

func newTracker(announcerUrlString, infoHash string, peerManager *peerManager) (*Scrapper, error) {
	var baseTracker *Scrapper
	var trackerStrategy scrapeStrategy
	trackerURL, err := url.Parse(announcerUrlString)

	if err != nil {
		return nil, err
	}

	baseTracker = &Scrapper{
		peerManager: peerManager,
		infoHash:    infoHash,
		trackerURL:  trackerURL,
		timer:       time.NewTimer(time.Nanosecond),
	}

	log.Printf("url: %v, url scheme : %v", announcerUrlString, trackerURL.Scheme)
	if trackerURL.Scheme == httpScheme {
		trackerStrategy = &httpTracker2{
			baseTracker,
		}
	} else {
		trackerStrategy = &httpTracker2{
			baseTracker,
		}
	}

	baseTracker.strategy = trackerStrategy

	log.Printf("creating new Scrapper: %v", reflect.TypeOf(baseTracker))

	return baseTracker, nil
}

func (t *Scrapper) getCurrentState() string {
	return ""
}

func (t *Scrapper) getAnnouncerUrl() *url.URL {
	return nil
}
func (t *Scrapper) getInfoHash() string {
	return ""
}
func (t *Scrapper) getTransferStats() (int, int, int) {
	return 0, 0, 0
}

func (t *Scrapper) startScrapper(ctx context.Context) {

	t.state = &initialRequest{scrapper: t}
	t.state.handle()
}

func (t *Scrapper) stopTracker() {
	t.state.cancel()
}
func (t *Scrapper) resetTracker(duration time.Duration) {
	t.state.reset(duration)
}

type trackerRequestState interface {
	handle()
	cancel()
	reset(duration time.Duration)
}

type initialRequest struct {
	scrapper *Scrapper
}

func (i *initialRequest) handle() {
	var _ error
	log.Printf("trackerUrl: %v", i.scrapper.trackerURL.String())
	log.Printf("sending initial 2 Scrapper request, trackerType : %v", reflect.TypeOf(i.scrapper))
	interval, err := i.scrapper.strategy.handleRequest()
	if err != nil {
		log.Fatalf("err:  %v", err)
		return
	}
	i.scrapper.interval = time.Duration(interval)
	i.scrapper.state = &recurringRequest{tracker: i.scrapper}
	log.Print("launching Scrapper request")
	i.scrapper.state.handle()

}

func (i *initialRequest) cancel() {

}

func (i *initialRequest) reset(time.Duration) {

}

type recurringRequest struct {
	tracker *Scrapper
}

func (r *recurringRequest) handle() {

	r.tracker.timer = time.AfterFunc(r.tracker.interval, func() {
		log.Printf("sending initial Scrapper request, trackerType : %v", reflect.TypeOf(r.tracker))

		interval, err := r.tracker.strategy.handleRequest()
		if err != nil {
			return
		}

		r.tracker.interval = time.Duration(interval)
		r.tracker.state = &recurringRequest{tracker: r.tracker}
		r.handle()
	})

}

func (r *recurringRequest) cancel() {
	r.tracker.timer.Stop()
	r.tracker.state = &initialRequest{}
}

func (r *recurringRequest) reset(duration time.Duration) {
	r.tracker.timer.Stop()
	r.tracker.timer.Reset(duration)
}
