package parser

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"
)


const (
	PeerIDLength = 20
)

func GetInfoHash(dict *Dict) string {

	// InnerStartingPosition leaves out the key

	startingPosition := dict.MapDict["info"].KeyInfo.InnerStartingPosition

	endingPosition := dict.MapDict["info"].KeyInfo.EndingPosition


	torrentFileString := ToBencode(dict)

	torrentFileByte := []byte(torrentFileString)

	infoBytes := torrentFileByte[startingPosition:endingPosition]

	b := sha1.Sum(infoBytes)
	bSlice := b[:]

	println(hex.EncodeToString(bSlice))

	return string(bSlice)

}

func GetRandomId() string {

	randomSeed := rand.New(rand.NewSource(time.Now().UnixNano()))
	peerIDRandom := randomSeed.Perm(PeerIDLength)

	fmt.Printf("%v", peerIDRandom)
	var peerIDRandomArr []byte
	var peerId string

	for _, n := range peerIDRandom {
		fmt.Printf("%v\n", n)
		peerIDRandomArr = append(peerIDRandomArr, byte(n))
	}

	x := (PeerIDLength * PeerIDLength) / hex.EncodedLen(PeerIDLength)

	peerIDRandomSha := sha1.Sum(peerIDRandomArr)
	peerIDRandomShaSlice := peerIDRandomSha[:x]
	peerId = hex.EncodeToString(peerIDRandomShaSlice)
	return peerId
}

func SaveTorrentFile(file []byte, fileName string) {
	homeDir, _ := os.UserHomeDir()

	dir := homeDir + "/DawnTorrent"

	_ = os.Mkdir(dir, os.ModeDir)
	fileName = dir + "/" + fileName
	_ = ioutil.WriteFile(fileName, file, 777)

}
