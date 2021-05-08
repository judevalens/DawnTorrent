package PeerProtocol

import (
	"testing"
)
func TestnewFileInfo(t *testing.T){
	//fileInfosSize := 1
	//fileInfos := make([]*fileProperty,fileInfosSize)
}

/*
func TestInitTorrentFile(t *testing.T) {
	torrent := new(Torrent)
	testTorrentFileSingleFile := initDownloader(torrent,"/home/jude/GolandProjects/DawnTorrent/files/ubuntu-20.04-desktop-amd64.iso.torrent")
	t.Run("test metadata 1 [single file]", func(t *testing.T) {
		announceTestValue := "https://torrent.ubuntu.com/announce"
		fileNameTestValue := "ubuntu-20.04-desktop-amd64.iso"
		infoHashHexTestValue := strings.ToLower("9FC20B9E98EA98B4A35E6223041A5EF94EA27809")
		fileLenTest := 2715254784
		filePieceLenTest := 1048576
		if testTorrentFileSingleFile.Announce !=	announceTestValue{
			t.Errorf("announcer address is incorrect : got %v, want %v", testTorrentFileSingleFile.Announce, announceTestValue)
		}
		if testTorrentFileSingleFile.Name != fileNameTestValue{
			t.Errorf("torrent name is incorrect : got -> %v, want -> %v", testTorrentFileSingleFile.Name, fileNameTestValue)
		}
		infoHashHex := hex.EncodeToString([]byte(testTorrentFileSingleFile.InfoHash))

	if strings.ToLower(infoHashHex) != infoHashHexTestValue {
		t.Errorf("infoHash is incorrect: got -> %v, want -> %v", strings.ToLower(infoHashHex), infoHashHexTestValue)
	}

	if testTorrentFileSingleFile.FileLength != fileLenTest {
		t.Errorf("file Len is incorrect: got -> %v, want -> %v", testTorrentFileSingleFile.FileLength, fileLenTest)
	}

		if testTorrentFileSingleFile.pieceLength != filePieceLenTest {
			t.Errorf("piece Len is incorrect: got -> %v, want -> %v", testTorrentFileSingleFile.pieceLength, filePieceLenTest)
		}

		// TODO add more extensive test for the metadata
	})
	t.Run("test file's info [single file] 1", func(t *testing.T) {
		filePathTest := "ubuntu-20.04-desktop-amd64.iso"
		fileStartIndexTest := 0
		fileEndIndexTest := 2715254784
		fileLengthTest := 2715254784
		fileIndexTest := 0

		if testTorrentFileSingleFile.FileProperties[0].Path!= filePathTest {
			t.Errorf("file path is incorrect : got -> %v, want -> %v", testTorrentFileSingleFile.FileProperties[0].Path,filePathTest)
		}
		if testTorrentFileSingleFile.FileProperties[0].StartIndex != fileStartIndexTest {
			t.Errorf("start App is incorrect : got -> %v, want -> %v", testTorrentFileSingleFile.FileProperties[0].StartIndex,fileStartIndexTest)
		}

		if testTorrentFileSingleFile.FileProperties[0].EndIndex != fileEndIndexTest {
			t.Errorf("End App is incorrect : got -> %v, want -> %v", testTorrentFileSingleFile.FileProperties[0].StartIndex,fileEndIndexTest)
		}

		if testTorrentFileSingleFile.FileProperties[0].BlockLength != fileLengthTest {
			t.Errorf("file Len is incorrect : got -> %v, want -> %v", testTorrentFileSingleFile.FileProperties[0].StartIndex, fileLengthTest)
		}

		if testTorrentFileSingleFile.FileProperties[0].FileIndex != fileIndexTest {
			t.Errorf("file App is incorrect : got -> %v, want -> %v", testTorrentFileSingleFile.FileProperties[0].StartIndex,fileIndexTest)
		}

	})

}
*/



