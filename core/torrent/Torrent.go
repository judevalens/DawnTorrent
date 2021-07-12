package torrent

import (
	"DawnTorrent/rpc/torrent_state"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"github.com/jackpal/bencode-go"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"path/filepath"
	"strings"
)

const (
	singleMode   = iota
	multipleMode = iota
)

type Torrent struct {
	Announce     string     `bencode:"announce"`
	AnnounceList []string   `bencode:"announce-list"`
	CreationDate int        `bencode:"creation date"`
	Comment      string     `bencode:"comment"`
	CreatedBy    string     `bencode:"created by"`
	Encoding     string     `bencode:"encoding"`
	SingleInfo   SingleInfo `bencode:"info"`
	Multiple     MultipleInfo
	InfoHashHex  string
	InfoHash     string
	FileMode     int
	FileSegments []FileSegment
	PieceLength  int
	Length       int
	Pieces       string
	serializedState *torrent_state.TorrentInfo
}

type SingleTorrent struct {
	Info SingleInfo `bencode:"info"`
}

type MultipleTorrent struct {
	Info MultipleInfo `bencode:"info"`
}

type Info interface {
}

type SingleInfo struct {
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type MultipleInfo struct {
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
	Name        string `bencode:"name"`
	Files       []File `bencode:"files"`
}

type File struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type FileSegment struct {
	File
	StartIndex int
	EndIndex   int
	parentName string
	mode       int
}

func (f FileSegment) GetPath() string {

	filePath := ""
	if f.mode == multipleMode {
		filePath = filepath.Join(filePath, f.parentName)

	}

	for _, p := range f.Path {
		filePath = filepath.Join(filePath, p)

	}

	return filepath.FromSlash(filePath)
}

func CreateNewTorrent(torrentPath string) (*Torrent, error) {

	file, err := ioutil.ReadFile(torrentPath)
	if err != nil {
		return nil, err
	}

	torrent := &Torrent{}
	err = bencode.Unmarshal(bytes.NewBuffer(file), torrent)

	if err != nil {
		return nil, err
	}

	/// doing some dirty work to calculate the infoHash of the torrent
	/// Basically we have to unmarshal the torrent file twice because we have no idea, what the info mode going to be (single or multiple files) :)
	infoBytes := bytes.NewBuffer([]byte{})

	if torrent.SingleInfo.Length != 0 {
		torrent.FileMode = singleMode

		info := &SingleTorrent{}

		err := bencode.Unmarshal(bytes.NewBuffer(file), info)
		if err != nil {
			return nil, err
		}

		err = bencode.Marshal(infoBytes, info.Info)
		if err != nil {
			return nil, err
		}

		logrus.Infof("hash: \n%v", string(infoBytes.Bytes()))
	} else {
		torrent.FileMode = multipleMode

		info := &MultipleTorrent{}

		err := bencode.Unmarshal(bytes.NewBuffer(file), info)
		if err != nil {
			return nil, err
		}
		err = bencode.Marshal(infoBytes, info.Info)

		torrent.Multiple = info.Info

		if err != nil {
			return nil, err
		}
	}

	torrent.InfoHash, torrent.InfoHashHex = calculateInfoHash(infoBytes.Bytes())

	torrent.buildFileSegment()

	logrus.Infof("InfoHash %v\nInfoHashhex %v", torrent.InfoHash, strings.ToUpper(torrent.InfoHashHex))

	return torrent, nil
}

func (torrent *Torrent) buildFileSegment() {
	fileSegments := make([]FileSegment, 0)
	fileLength := 0

	if torrent.FileMode == multipleMode {

		startIndex := 0
		endIndex := 0
		for fileIndex, file := range torrent.Multiple.Files {
			if fileIndex == 0 {
				endIndex = file.Length
			} else {
				startIndex = fileSegments[fileIndex-1].EndIndex
				endIndex = startIndex + file.Length
			}

			fileSegments = append(fileSegments, FileSegment{
				mode:       torrent.FileMode,
				parentName: torrent.Multiple.Name,
				StartIndex: startIndex,
				EndIndex:   endIndex,
				File:       file,
			})

			fileLength += file.Length
			torrent.PieceLength = torrent.Multiple.PieceLength
			torrent.Pieces = torrent.Multiple.Pieces
		}
	} else {
		fileSegments = append(fileSegments, FileSegment{
			mode:       torrent.FileMode,
			parentName: torrent.SingleInfo.Name,
			StartIndex: 0,
			EndIndex:   torrent.SingleInfo.Length,
			File: File{
				Length: torrent.SingleInfo.Length,
				Path:   []string{torrent.SingleInfo.Name},
			},
		})
		fileLength = torrent.SingleInfo.Length
		torrent.PieceLength = torrent.SingleInfo.PieceLength
		torrent.Pieces = torrent.SingleInfo.Pieces
	}

	torrent.FileSegments = fileSegments
	torrent.Length = fileLength
}

func (torrent *Torrent) Serialize() *torrent_state.TorrentInfo {
	if torrent.serializedState != nil{
		return torrent.serializedState
	}
	mode := torrent_state.TorrentInfo_Single
	if torrent.FileMode == multipleMode {
		mode = torrent_state.TorrentInfo_Multiple
	}
	paths := make([]*torrent_state.FilePath,len(torrent.FileSegments))
	for i, segment := range torrent.FileSegments {
		paths[i] = &torrent_state.FilePath{
			Path:   segment.GetPath(),
			Length: int32(segment.Length),
		}
	}

	torrent.serializedState = &torrent_state.TorrentInfo{
		Mode:          mode,
		TorrentLength: int32(torrent.Length),
		Infohash:      torrent.InfoHashHex,
		Paths:         paths,
	}

	return 	torrent.serializedState
}

func calculateInfoHash(info []byte) (string, string) {

	// InnerStartingPosition leaves out the key
	infoHash := sha1.Sum(info)
	infoHashSlice := infoHash[:]
	//println(hex.EncodeToString(infoHashSlice))

	return string(infoHash[:]), hex.EncodeToString(infoHashSlice)

}
