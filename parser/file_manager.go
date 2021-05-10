package parser

import (
	"DawnTorrent/utils"
	"io/ioutil"
	"os"
)


const (
)
func SaveTorrentFile(file []byte, fileName string) {
	_ = os.Mkdir(utils.TorrentHomeDir, os.ModeDir)
	fileName = utils.TorrentHomeDir + "/" + fileName
	_ = ioutil.WriteFile(fileName, file, 777)

}
