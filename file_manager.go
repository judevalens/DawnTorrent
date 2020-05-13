package main

import (
	"crypto/sha1"
	"encoding/hex"
	"math/rand"
	"strconv"
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

func GetInfoHash(file []byte, dict *parser.Dict) string {


	// InnerStartingPosition leaves out the key

	startingPosition := dict.MapDict["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.MapDict["info"].KeyInfo.EndingPosition

	infoBytes := file[startingPosition:endingPosition]


	b := sha1.Sum(infoBytes)
	bSlice := b[:]




	return hex.EncodeToString(bSlice)

}

func GetHash(dict *parser.Dict)  [][]byte{
	bytesPerHash := 20

	hash := make([][]byte,0)

	s := dict.MapDict["info"].MapString["pieces"]

	sLen := len(s)

	counter  := 0
	for counter < sLen{
		b :=  []byte(s[counter:counter+bytesPerHash])
		hash = append(hash, b)
		counter += bytesPerHash

	}
	return hash
}

func getPeerId() string {
	peerIDRandom := rand.Perm(PeerIDLength)
	var peerId string
	for _,n  := range peerIDRandom{
		peerId += strconv.Itoa(n)
	}
	return peerId
}

func buildTrackerRequest(){

}

func makeTrackerRequest(){

}