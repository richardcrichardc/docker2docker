package main

import "io"

/*
	MeteredReader wrap another reader and maintains a count of data read through
	the reader.
*/

type MeteredReader struct {
	reader         io.Reader
	bytesReadCount *int64
}

func NewMeteredReader(reader io.Reader, bytesReadCount *int64) io.Reader {
	return &MeteredReader{reader, bytesReadCount}
}

func (mr *MeteredReader) Read(p []byte) (n int, err error) {
	n, err = mr.reader.Read(p)
	*mr.bytesReadCount += int64(n)
	return
}
