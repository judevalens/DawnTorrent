package main

import (
  "bufio"
  "fmt"
  "torrent/parser"
  "torrent/utils"
)

func main() {


  torrentPath := "files/ubuntu-20.04-desktop-amd64.iso.torrent"

  torrent := NewTorrent(torrentPath)

  torrent.addPeers()

  conn , err := torrent.Peers[5].connectTo()

  fmt.Printf("%v\n",err)

  answer := make([]byte,0)
  _, writeErr := conn.Write(parser.GetMsg(torrent.TorrentFile.MapString["infoHash"], utils.MyID, parser.HanShakeMsg))

    fmt.Printf("writeErr %v\n",writeErr)



    status, err := bufio.NewReader(conn).Read(answer)

    //fmt.Printf("ANSWER %v\n", answer)
    fmt.Printf("stat %v err %v addr %v\n", status, err, conn.LocalAddr().String())


  Listen()


//  SaveTorrentFile([]byte(parser.ToBencode(dict)), filename)
 // handShake("","")
  //println(LocalAddress())



}
