package main


import (
    "bytes"
    bencode "github.com/jackpal/bencode-go"
)

import (
 "fmt"
 //"log"
)
//import "bytes"

type myAwesomeObject struct {
  announce string
}



func main() {
  fmt.Printf("TORRENT")
  b := readFile("files/tears-of-steel.torrent")

  //fmt.Printf("%v\n",string(b))




  data2, _ := bencode.Decode(bytes.NewReader(b))

  fmt.Printf("%v",data2)



  data := myAwesomeObject{}
  err := bencode.Unmarshal(bytes.NewReader(b), &data)
  fmt.Printf("%v",data)

  if err != nil {
  }

}