package main

import (
  "strings"
  "torrent/parser"
)

func main() {


  torrentPath := "files/ubuntu-20.04-desktop-amd64.iso.torrent"

  torrentPaths := strings.Split(torrentPath, "/")

  filname := torrentPaths[len(torrentPaths)-1]

  var dict = parser.Unmarshall(torrentPath)


  SaveTorrentFile([]byte(parser.ToBencode(dict)),filname)


}