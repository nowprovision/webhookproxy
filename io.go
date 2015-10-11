package webhookproxy

import "io"
import "errors"

// todo: is 1024bytes this a good default chunk size?
const chunkSize int64 = 1024

func CopyMax(maxTransfer int64, w io.Writer, r io.Reader) (written int64, err error) {

	adjChunkSize := chunkSize

	if maxTransfer > 0 {
		if maxTransfer < chunkSize {
			adjChunkSize = maxTransfer
		}
		written, err = io.CopyN(w, r, int64(adjChunkSize))
	} else {
		return 0, errors.New("Payload exceed max transfer size")
	}

	if err == nil {
		return CopyMax(maxTransfer-written, w, r)
	} else {
		if err == io.EOF {
			return written, nil
		} else {
			return written, err
		}
	}
}
