package webhookproxy

import "io"
import "bytes"

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

func NewStringPayload(payload string) io.ReadCloser {
	return &nopCloser{bytes.NewBufferString(payload)}
}
