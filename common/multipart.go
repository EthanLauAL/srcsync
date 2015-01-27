package common

import (
	"encoding/binary"
	"io"
)

type MultipartReader struct {
	reader io.Reader
}

func NewMultipartReader(r io.Reader) *MultipartReader {
	return &MultipartReader{r}
}

func (r *MultipartReader) NextPart() (io.Reader,error) {
	var size uint32
	err := binary.Read(r.reader, binary.LittleEndian, &size)
		if err != nil { return nil, err }
	return io.LimitReader(r.reader, int64(size)),nil
}

type MultipartWriter struct {
	writer io.Writer
}

func NewMultipartWriter(w io.Writer) *MultipartWriter {
	return &MultipartWriter{w}
}

func (w *MultipartWriter) WritePart(bytes []byte) error {
	err := binary.Write(w.writer, binary.LittleEndian, uint32(len(bytes)))
		if err != nil { return err }
	_,err = w.writer.Write(bytes)
	return err
}
