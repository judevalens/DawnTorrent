package app

import (
	"DawnTorrent/utils"
	"os"
	"path"
)

type PieceWriter struct {
	currentPiece    *Piece
	buffer          []byte
	currenBufferLen int
	isBufferFull    bool
	pieceLength     int // represent the normal length of each piece (last piece's length might be different)
	metaData 		[]fileMetadata
}

func (writer *PieceWriter) setNewPiece(currentPiece *Piece)  {
		writer.currentPiece = currentPiece
		writer.buffer = make([]byte,currentPiece.pieceLength)
		writer.isBufferFull = currentPiece.downloaded == currentPiece.pieceLength
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
	writer.isBufferFull  = writer.currenBufferLen == len(writer.buffer)
	return writer.isBufferFull
}
/*
	writes a chunk of data to a file.
*/
func (writer *PieceWriter) writeToFile(data []byte, fileName string, startIndex int) error {

	filePath := path.Join(utils.TorrentHomeDir, fileName)
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	_, err = file.WriteAt(data, int64(startIndex))
	if err != nil {
		return err
	}
	return nil
}

/*
	writes a completed piece to a file
*/
func (writer *PieceWriter) writePiece() error {
	/*
		we must know the index at which to write a piece in a file
		we have check for boundaries so we can determine if a chunk of data overlaps over multiple files.
	*/
	absStartIndex := writer.currentPiece.PieceIndex * writer.pieceLength

	absEndIndex := absStartIndex + writer.currentPiece.pieceLength
	relativeStartIndex := 0
	relativeEndIndex :=writer.currentPiece.pieceLength
	for _, metadatum := range writer.metaData {
		if absStartIndex <= metadatum.StartIndex{
			if absEndIndex > metadatum.EndIndex{
				relativeEndIndex = writer.currentPiece.pieceLength - (absEndIndex-metadatum.EndIndex)
			}

			err := writer.writeToFile(writer.buffer[relativeStartIndex:relativeEndIndex], metadatum.Path, absStartIndex)
			if err != nil {
				return err
			}

			relativeStartIndex = relativeEndIndex
		}
	}
	return nil
}
