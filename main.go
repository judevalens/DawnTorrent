package main

//import "bytes"

type myAwesomeObject struct {
  announce string
}



func main() {
  b := readFile("files/big-buck-bunny.torrent")

  ///Decode(b,0)


  /*var field byte
  var pos = 1

  for field == 0{


    s,position,f := getField(b,pos,'d')

    field = f
    fmt.Printf("%v\n",s)
    fmt.Printf("position %v\n",position)
    fmt.Printf("field %s\n",string(field))

    pos = position
  }
*/
  dict := new(Dict)

  parse(dict,b,0)









}