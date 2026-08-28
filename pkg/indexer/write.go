package indexer

import (
	"bufio"
	"io"
)

// Writer receives FileEntry values on a channel and streams NDJSON to w.
type Writer struct {
	w    *bufio.Writer
	ch   chan FileEntry
	done chan error
}

// NewWriter creates a streaming NDJSON writer.
// Call Send to enqueue entries and Close when done.
func NewWriter(w io.Writer, bufSize int) *Writer {
	if bufSize <= 0 {
		bufSize = 256
	}
	wr := &Writer{
		w:    bufio.NewWriterSize(w, 64*1024),
		ch:   make(chan FileEntry, bufSize),
		done: make(chan error, 1),
	}
	go wr.loop()
	return wr
}

func (wr *Writer) loop() {
	var writeErr error
	for e := range wr.ch {
		if writeErr != nil {
			continue // drain channel
		}
		b, err := e.MarshalNDJSON()
		if err != nil {
			writeErr = err
			continue
		}
		if _, err = wr.w.Write(b); err != nil {
			writeErr = err
			continue
		}
		if err = wr.w.WriteByte('\n'); err != nil {
			writeErr = err
		}
	}
	if writeErr == nil {
		writeErr = wr.w.Flush()
	}
	wr.done <- writeErr
}

// Send enqueues a FileEntry for writing.
func (wr *Writer) Send(e FileEntry) { wr.ch <- e }

// Close signals no more entries and waits for the writer to finish.
// Returns the first write error encountered, if any.
func (wr *Writer) Close() error {
	close(wr.ch)
	return <-wr.done
}
