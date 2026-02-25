package services

import "bytes"

// bytesReader wraps a byte slice in a bytes.Reader for use with excelize.OpenReader.
func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
