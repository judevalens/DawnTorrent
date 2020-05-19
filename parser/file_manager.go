package parser

import (
	"io/ioutil"
	"os"
	"torrent/utils"
)


const (
)



func SaveTorrentFile(file []byte, fileName string) {
	_ = os.Mkdir(utils.DawnTorrentHomeDir, os.ModeDir)
	fileName = utils.DawnTorrentHomeDir + "/" + fileName
	_ = ioutil.WriteFile(fileName, file, 777)

}
