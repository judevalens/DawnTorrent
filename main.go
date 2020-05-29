package main

import (
	//"time"
	"torrent/PeerProtocol"
)

func main() {
	done := make(chan bool)

	torrentPath := "files/ubuntu-20.04-live-server-amd64.iso.torrent"
	torrent := PeerProtocol.NewTorrent(torrentPath, done)
	//go torrent.RequestQueueManager()
	_ = torrent
	<-done
	/*
		t := new(PeerProtocol.TestQueue)

		t.Q = PeerProtocol.NewQueue(t)

		for i := 0; i < 500; i++{
			t.Q.Add(i)
		}

		t.Q.Run(40)

		time.AfterFunc(time.Second, func() {
			for i := 500; i < 750; i++{
				t.Q.Add(i)
			}
		})

		time.AfterFunc(2*time.Second, func() {
			for i := 750; i <= 1200; i++{
				t.Q.Add(i)
			}
		})


		time.AfterFunc(5*time.Second, func() {
			for i := 750; i <= 100000; i++{
				t.Q.Add(i)
			}
		})


		time.AfterFunc(8*time.Second, func() {

			go func() {
				for i := 100000; i <= 300099; i++{
					t.Q.Add(i)
				}
			}()

			go func() {
				for i := 300099; i <= 500000; i++{
					t.Q.Add(i)
				}
			}()


		})

		d := 0
		for{
			d++
		}
	*/
}
