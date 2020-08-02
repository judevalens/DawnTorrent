package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	upnp "github.com/huin/goupnp/dcps/internetgateway1"
)

const DEBUG = true
const PORT = 6881
const PORT2 = 6881
const UpFlag = "up"

// path types
const (
	PieceHashPath   = 0
	TorrentDataPath = 1
	DownloadedFile  = 2
	Ipc = 3
)

var LocalAddr, _ = net.ResolveTCPAddr("tcp", LocalAddress().String()+":"+strconv.Itoa(PORT2))
var LocalAddr2, _ = net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(PORT))
var MyID = GetRandomId()

var KeepAliveDuration, _ = time.ParseDuration("120s")

var homeDir, _ = os.UserHomeDir()

var TorrentHomeDir = filepath.FromSlash(homeDir + "/DawnTorrent/files")

var SavedTorrentDir = filepath.FromSlash(homeDir + "/DawnTorrent/torrents")

func GetRandomId() string {
	PeerIDLength := 20

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

func Debugln(st string) {

	if DEBUG {
		fmt.Println(st)
	}
}

func LocalAddress() net.IP {
	list, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, iface := range list {

		fmt.Printf("%v flag %v index %v\n", iface.Name, iface.Flags.String(), iface.Index)

		interfaceAddrs, _ := iface.Addrs()
		for _, addr := range interfaceAddrs {
			ip, _, _ := net.ParseCIDR(addr.String())

			flags := strings.Split(iface.Flags.String(), "|")

			if !ip.IsLoopback() {
				addrArr, _ := iface.Addrs()

				for _, flag := range flags {
					if flag == "up" {
						fmt.Printf("ip %v addr %v\n", addrArr[len(addrArr)-1].String(), ip)
						if ip.To4() != nil {
							return ip
						}

					}
				}

			}

		}

	}

	return net.IP{}
}

// Mask a bit at given position
//
// 1 turns bit on | 0 turns bit off
func BitMask(b uint8, bits []int, action int) uint8 {

	if action == 1 {
		for _, v := range bits {
			bitPosition := uint8(math.Exp2(float64(v)))
			b = b | bitPosition
		}
	} else if action == 0 {
		for _, v := range bits {
			bitPosition := uint8(math.Exp2(float64(v)))
			bitPositionFlipped := flipBit(bitPosition)
			b = b & bitPositionFlipped
		}
	}

	return b

}
func IsBitOn(b uint8, pos int) bool {
	bitPosition := uint8(math.Exp2(float64(pos)))
	a := b & bitPosition

	return a != 0
}
func flipBit(b uint8) uint8 {
	a := uint8(255)
	a = a &^ b
	return a
}

func GetPath(pathType int, path string, fileName string) string {
	var path2 string
	var pathRoot string
	switch pathType {
	case PieceHashPath:
		pathRoot = filepath.FromSlash(SavedTorrentDir + "/" + path)

		if _, err := os.Stat(pathRoot); os.IsNotExist(err) {
			_ = os.MkdirAll(pathRoot, os.ModePerm)

		}
		path2 = filepath.FromSlash(SavedTorrentDir + "/" + path + "/" + fileName)
	case TorrentDataPath:
		pathRoot = filepath.FromSlash(SavedTorrentDir + "/" + path)

		if _, err := os.Stat(pathRoot); os.IsNotExist(err) {
			_ = os.MkdirAll(pathRoot, os.ModePerm)

		}
		path2 = filepath.FromSlash(SavedTorrentDir + "/" + path + "/" + fileName)
	case DownloadedFile:
		pathRoot = filepath.FromSlash(TorrentHomeDir + "/" + path)

		if _, err := os.Stat(pathRoot); os.IsNotExist(err) {
			_ = os.MkdirAll(pathRoot, os.ModePerm)

		}
		path2 = filepath.FromSlash(TorrentHomeDir + "/" + path + "/" + fileName)
	case Ipc:
		pathRoot = filepath.FromSlash(TorrentHomeDir + "/" + path)
		if _, err := os.Stat(pathRoot); os.IsNotExist(err) {
			_ = os.MkdirAll(pathRoot, os.ModePerm)

		}
		path2 = filepath.FromSlash(TorrentHomeDir + "/" + path )

	}
	return path2
}

func GetFileName(path string) string {
	extensionLen := len(filepath.Ext(path))
	fileNameLen := len(path)

	return path[:fileNameLen-extensionLen]
}

/// That's a work in progress
/// for now I will manually forward the port

func forwardPort(port string) {

	connectionClient, _, err := upnp.NewWANIPConnection1Clients()

	if err != nil {

		log.Fatal(err)

	}

	ip := connectionClient[0].AddPortMapping("", PORT, "tcp", PORT, LocalAddress().String(), true, "for torrent 2", 100000)

	fmt.Printf("len %v name %v\n", len(connectionClient), ip)

}
