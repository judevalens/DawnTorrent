package main

func main() {


  torrentPath := "files/ubuntu-20.04-live-server-amd64.iso.torrent"

  torrent := NewTorrent(torrentPath)


 go torrent.Peers[4].connectTo(torrent)

  torrent.Listen()




}
