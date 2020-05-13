package main

import (
  "fmt"
  "torrent/parser"
)

func main() {


  var dict = parser.Unmarshall("files/tears-of-steel.torrent")
//
 // file := parser.ReadFile("files/tears-of-steel.torrent")

  println("DONE !!!!!!!!!!!!")
  fmt.Printf("%#v\n",dict.DataList[0])
  fmt.Printf("%#v\n",dict.MapDict["info"].KeyInfo)
  fmt.Printf("%#v\n",dict.MapList["announce-list"].KeyInfo)

 //fmt.Printf("%v\n", string(file[dict.MapDict["info"].KeyInfo.InnerStartingPosition:dict.MapDict["info"].KeyInfo.InnerEndingPosition]))

 b:= GetInfoHash(dict.OriginalFile,dict)


 fmt.Printf("%s", b)

}