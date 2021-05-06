package protocol

import (
	"DawnTorrent/protocol/scrapper"
	"context"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)


func TestNewTorrent(t *testing.T) {
	testTorrent := "../files/ubuntu-20.04-desktop-amd64.iso.torrent"
	torrent, err := createNewTorrent(testTorrent)

	if err != nil{
		assert.Fail(t, err.Error())
		return
	}

	name := "ubuntu-20.04-desktop-amd64.iso"
	announcerUrl := "https://torrent.ubuntu.com/announce"
	announcerUrlList := []string{"https://torrent.ubuntu.com/announce","https://ipv6.torrent.ubuntu.com/announce"}
	infoHash := "9fc20b9e98ea98b4a35e6223041a5ef94ea27809"
	torrentSize := "2715254784"
	var tests = []struct{
		expected,actual string
	}{
		{name,torrent.Name},
		{announcerUrl,torrent.AnnouncerUrl},
		{announcerUrlList[0],torrent.AnnounceList[0]},
		{announcerUrlList[1],torrent.AnnounceList[1]},
		{infoHash,torrent.InfoHashHex},
		{torrentSize,strconv.Itoa(torrent.FileLength)},
	}

	for _, test := range  tests{
		assert.Equal(t, test.expected,test.actual, "should be equal")
	}
}

 func TestTorrentManager_Init(t *testing.T) {
	 testTorrent := "../files/ubuntu-20.04-desktop-amd64.iso.torrent"

	 _ = NewTorrentManager(testTorrent)
	 //manager.Init()
 }

func TestTracker(t *testing.T){

	var err error
	testTorrent := "../files/ubuntu-20.04-desktop-amd64.iso.torrent"
	torrent, err := createNewTorrent(testTorrent)

	if err != nil{
		assert.Fail(t, err.Error())
		return
	}

	peerManager := newPeerManager(nil,torrent.InfoHashHex)


	_ = scrapper.newTracker(torrent.AnnouncerUrl,torrent.InfoHashHex,peerManager)
	_ = context.Background()

}