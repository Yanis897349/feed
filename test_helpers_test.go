package main

import (
	"io"
	"net/http"
	"sync"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type failingWriter struct {
	err error
}

func (writer failingWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

type blockingReader struct {
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (reader *blockingReader) Read([]byte) (int, error) {
	close(reader.started)
	<-reader.release
	return 0, io.EOF
}

func (reader *blockingReader) Cancel() bool {
	reader.once.Do(func() { close(reader.release) })
	return true
}

func (reader *blockingReader) Close() error {
	reader.Cancel()
	return nil
}
