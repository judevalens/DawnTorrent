package main

import (
  "encoding/hex"
  "fmt"
  "torrent/parser"
)

//import "bytes"

type myAwesomeObject struct {
  announce string
}



func main() {


  var dict *parser.Dict = parser.Unmashal("files/FreeBSD-11.4-BETA1-i386-disc1.torrent")

  println("DONE !!!!!!!!!!!!")
  fmt.Printf("%#v\n",dict)
  println("HASH")
  hashes := parser.GetHash(dict)

  fmt.Printf("%#v\n",hashes)


  file := parser.ReadFile("files/FreeBSD-11.4-BETA1-i386-disc1.torrent")

  _ = file



  encodedStr := hex.EncodeToString(hashes[0])

  fmt.Printf("%s\n", encodedStr)

}