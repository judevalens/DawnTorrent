package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
	"torrent/parser"
)

type trackerRequest struct {
	infoHash []byte
	peerId string
	ip string
	port int
	uploaded int
	downloaded int
	left int
	event int
}


const (
	PeerIDLength = 20
)

func GetInfoHash(dict *parser.Dict) string {

	// InnerStartingPosition leaves out the key

	startingPosition := dict.MapDict["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.MapDict["info"].KeyInfo.EndingPosition

	infoBytes := dict.OriginalFile[startingPosition:endingPosition]


	b := sha1.Sum(infoBytes)
	bSlice := b[:]

	println(hex.EncodeToString(bSlice))


	return string(bSlice)

}


func getRandomId() string {

	randomSeed  := rand.New(rand.NewSource(time.Now().UnixNano()))
	peerIDRandom := randomSeed.Perm(PeerIDLength)

	fmt.Printf("%v",peerIDRandom)
	var peerIDRandomArr []byte
	var peerId string

	for _,n  := range peerIDRandom{
		fmt.Printf("%v\n", n)
		peerIDRandomArr =  append(peerIDRandomArr,byte(n))
	}

	x := (PeerIDLength*PeerIDLength)/hex.EncodedLen(PeerIDLength)

	peerIDRandomSha := sha1.Sum(peerIDRandomArr)
	peerIDRandomShaSlice := peerIDRandomSha[:x]
	peerId = hex.EncodeToString(peerIDRandomShaSlice)
	return peerId
}

func GetPeersInfo(dict *parser.Dict) *parser.Dict{

	PeerId := getRandomId()
	println(len(PeerId))
	infoHash := GetInfoHash(dict)
	port:= "6881"
	uploaded := 0
	downloaded := 0
	left, _ := strconv.Atoi(dict.MapDict["info"].MapString["length"])
	event := "started"
	tackerParams := "info_hash="+infoHash+"&peer_id="+PeerId+"&port="+port+"&uploaded="+strconv.Itoa(uploaded)+"&downloaded="+strconv.Itoa(downloaded)+"&left="+strconv.Itoa(left)+"&event="+event

	tackerParams = url.PathEscape(tackerParams)


	trackerUrl := dict.MapString["announce"] +"?" + tackerParams
	//fmt.Printf("%s\n",trackerUrl)

	trackerResponseByte, _ := http.Get(trackerUrl)
	trackerResponse,_ := ioutil.ReadAll(trackerResponseByte.Body)

	fmt.Printf("%v\n",string(trackerResponse))

	trackerDictResponse := parser.UnmarshallFromArray(trackerResponse)

	/*
	//_ =trackerDictResponse
	//fmt.Printf("%v port %v peer id %v\n", trackerDictResponse.MapList["peers"].LDict[12].MapString["ip"], trackerDictResponse.MapList["peers"].LDict[12].MapString["port"],trackerDictResponse.MapList["peers"].LDict[12].MapString["peer id"])
	fmt.Printf("%v\n",trackerDictResponse.KeyInfo)
	fmt.Printf("%v\n",trackerDictResponse.DataList[0])
	fmt.Printf("%v\n",trackerDictResponse.DataList[1])
	fmt.Printf("%v\n",trackerDictResponse.DataList[2])
	fmt.Printf("%v\n",trackerDictResponse.MapList["peers"].DataList[1].Index)
	fmt.Printf("%s\n",trackerDictResponse.MapString)

	println("TO STRING")

	toS := parser.ToString(trackerDictResponse)

	trackerDictResponseTwin := parser.UnmarshallFromArray([]byte(toS))

	fmt.Printf("%s\n",trackerDictResponse.MapList["peers"].LDict[12].MapString["peer id"])
	fmt.Printf("%s\n",trackerDictResponseTwin.MapList["peers"].LDict[12].MapString["peer id"])
*/

return trackerDictResponse
}

func SaveTorrentFile(file []byte,fileName string){
	homeDir, _ := os.UserHomeDir()

	dir := homeDir + "/DawnTorrent"

	_ = os.Mkdir(dir, os.ModeDir)
	fileName = dir +"/"+fileName
	_ = ioutil.WriteFile(fileName, file,777)


}