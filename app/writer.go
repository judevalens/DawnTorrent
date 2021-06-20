package app

import (
	"DawnTorrent/utils"
	"log"
	"os"
	"path"
)

type PieceWriter struct {
	currentPiece    *Piece
	buffer          []byte
	pieceIndex      int
	bufferSize      int
	currenBufferLen int
	isBufferFull    bool
	pieceLength     int // represent the normal length of each piece (last piece's length might be different)
	metaData        []fileMetadata
}

func (writer *PieceWriter) setNewPiece(bufferSize, pieceIndex int) {
	writer.pieceIndex = pieceIndex
	writer.bufferSize = bufferSize
	writer.buffer = make([]byte, bufferSize)
	writer.currenBufferLen = 0
	writer.isBufferFull = writer.currenBufferLen == writer.bufferSize
}

/*
	writes the subPieces to a buffer until the piece is completed
*/
func (writer *PieceWriter) writeToBuffer(data []byte, startIndex int) bool {
	if writer.currenBufferLen < len(writer.buffer) {
		subPieceLen := len(data)
		copy(writer.buffer[startIndex:startIndex+subPieceLen], data)
		writer.currenBufferLen += subPieceLen

	}
	writer.isBufferFull = writer.currenBufferLen == len(writer.buffer)
	return writer.isBufferFull
}

/*
	writes a chunk of data to a file.
*/
func (writer *PieceWriter) writeToFile(data []byte, fileName string, startIndex int) (int, error) {

	filePath := path.Join(utils.TorrentHomeDir, fileName)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return 0, err
	}
	nByte, err := file.WriteAt(data, int64(startIndex))
	log.Printf("wrote %v bytes", nByte)
	if err != nil {
		return 0, err
	}
	return nByte, nil
}

/*
	writes a completed piece to a file
*/
func (writer *PieceWriter) writePiece() error {
	/*
		we must know the index at which to write a piece in a file
		we have check for boundaries so we can determine if a chunk of data overlaps over multiple files.
	*/
	absStartIndex := writer.pieceIndex * writer.bufferSize
	writeAt := 0
	absEndIndex := absStartIndex + writer.bufferSize
	relativeStartIndex := 0
	relativeEndIndex := writer.bufferSize
	remaining := writer.bufferSize

	for _, metadatum := range writer.metaData {

		if remaining == 0 {
			break
		}

		log.Printf("file path %v, start Index %v, end index %v", metadatum.Path, metadatum.StartIndex, metadatum.EndIndex)

		/*	case a -> file is inside piece
			1---|-2----3--|--4
		*/
		caseA := metadatum.StartIndex > absStartIndex && metadatum.EndIndex < absEndIndex
		/*	case b -> piece overlaps over two files, trying to get first block
			1--|--2-|---3----4
		*/
		caseB := metadatum.StartIndex <= absStartIndex && metadatum.EndIndex > absStartIndex
		/*	case c -> piece overlaps over two file, trying to get second block
			1---|-2--|--3----4
		*/
		caseC := metadatum.StartIndex >= absStartIndex && metadatum.EndIndex >= absEndIndex

		if caseA {
			writeAt = 0
			relativeStartIndex = metadatum.StartIndex - absStartIndex
			relativeEndIndex = relativeStartIndex + metadatum.Length
		} else if caseB {
			// where to start writing a piece
			writeAt = absStartIndex - metadatum.StartIndex
			// if a piece extend over multiple files, we find the end index in the piece for that file
			if absEndIndex > metadatum.EndIndex {
				relativeEndIndex = writer.bufferSize - (absEndIndex - metadatum.EndIndex)
			}

		} else if caseC {
			writeAt = 0
			relativeStartIndex = metadatum.StartIndex - absStartIndex
			relativeEndIndex = writer.bufferSize
		}

		log.Printf("start index %v, abs start index %v, relative end index %v,abs end index %v, end index %v, path %v", relativeStartIndex, absStartIndex, relativeEndIndex, absEndIndex, metadatum.EndIndex, metadatum.Path)

		// write to slice of data to the piece where it belongs
		nByte, err := writer.writeToFile(writer.buffer[relativeStartIndex:relativeEndIndex], metadatum.Path, writeAt)
		if err != nil {
			return err
		}

		remaining -= nByte

		//we will start slice the piece for the next file at the end index where we sliced it for the previous file
		relativeStartIndex = relativeEndIndex
		relativeEndIndex = writer.bufferSize

	}
	return nil
}
