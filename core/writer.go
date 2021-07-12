package core

import (
	"DawnTorrent/core/torrent"
	"DawnTorrent/utils"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
)

/*
	writes a chunk of data to a file.
*/
func writeToFile(data []byte, filePath string, startIndex int) (int, error) {

	//create parent dir if it doesnt exist
	finalPath := filepath.Join(utils.TorrentHomeDir, filePath)
	parentDir := path.Dir(filepath.ToSlash(finalPath))
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		_ = os.MkdirAll(parentDir, os.ModePerm)
	}

	file, err := os.OpenFile(finalPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return 0, err
	}
	nByte, err := file.WriteAt(data, int64(startIndex))
	if err != nil {
		log.Panicf("could not write file\n%v", err.Error())
		return 0, err
	}
	return nByte, nil
}

/*
	writes a completed piece to disk
*/
func writePiece(piece *Piece, segments []torrent.FileSegment) error {
	/*
		we must know the index at which to write a piece in a file
		we have check for boundaries so we can determine if a chunk of data overlaps over multiple files.
	*/
	absStartIndex := piece.pieceStartIndex
	writeAt := 0
	absEndIndex := absStartIndex + piece.pieceLength
	relativeStartIndex := 0
	relativeEndIndex := piece.pieceLength
	remaining := piece.pieceLength

	for _, metadatum := range segments {

		if remaining == 0 {
			break
		}

		log.Debugf("file path %v, start Index %v, end index %v", metadatum.Path, metadatum.StartIndex, metadatum.EndIndex)

		/*	case a -> file is inside piece
			1---|-2----3--|--4
		*/
		caseA := absStartIndex < metadatum.StartIndex && absEndIndex > metadatum.EndIndex

		/*	case b -> piece overlaps over two files, trying to get first block
			1--|--2-|---3----4
		*/
		caseB := absStartIndex >= metadatum.StartIndex && absStartIndex <= metadatum.EndIndex

		/*	case c -> piece overlaps over two file, trying to get second block
			1---|-2--|--3----4
		*/
		caseC := absStartIndex <= metadatum.StartIndex && absEndIndex <= metadatum.EndIndex

		if caseA {
			writeAt = 0
			relativeStartIndex = metadatum.StartIndex - absStartIndex
			relativeEndIndex = relativeStartIndex + metadatum.Length
		} else if caseB {
			// where to start writing a piece
			writeAt = absStartIndex - metadatum.StartIndex
			// if a piece extend over multiple files, we find the end index in the piece for that file
			if absEndIndex > metadatum.EndIndex {
				relativeEndIndex = piece.pieceLength - (absEndIndex - metadatum.EndIndex)
			}

		} else if caseC {
			writeAt = 0
			relativeStartIndex = metadatum.StartIndex - absStartIndex
			relativeEndIndex = piece.pieceLength
		} else {

			continue

		}

		log.Debugf("start index %v, abs start index %v, relative end index %v,abs end index %v, end index %v, path %v", relativeStartIndex, absStartIndex, relativeEndIndex, absEndIndex, metadatum.EndIndex, metadatum.Path)

		// write to slice of data to the piece where it belongs
		nByte, err := writeToFile(piece.buffer[relativeStartIndex:relativeEndIndex], metadatum.GetPath(), writeAt)
		if err != nil {
			return err
		}

		remaining -= nByte

		//we will start slice the piece for the next file at the end index where we sliced it for the previous file
		relativeStartIndex = relativeEndIndex
		relativeEndIndex = piece.pieceLength

	}

	return nil
}
