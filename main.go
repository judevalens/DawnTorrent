package main

func main() {


  torrentPath := "files/ubuntu-20.04-live-server-amd64.iso.torrent"

  torrent := NewTorrent(torrentPath)


  _ , _ = torrent.Peers[0].connectTo(torrent)

 ////// torrent.Listen()




}
