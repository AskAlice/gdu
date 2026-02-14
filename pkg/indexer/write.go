package indexer

import (
	"bufio"
	"io"
	"sync"
)

// Writer writes index entries to a file
type Writer struct {
	writer  *bufio.Writer
	entryCh chan *FileEntry
	errorsCh chan error
	doneCh   chan struct{}
	wg       sync.WaitGroup
}

// NewWriter creates a new Writer
func NewWriter(w io.Writer) *Writer {
	writer := &Writer{
		writer:   bufio.NewWriter(w),
		entryCh:  make(chan *FileEntry, 100),
		errorsCh: make(chan error, 10),
		doneCh:   make(chan struct{}),
	}
	writer.wg.Add(1)
	go writer.run()
	return writer
}

func (w *Writer) run() {
	defer w.wg.Done()
	defer close(w.doneCh)

	for entry := range w.entryCh {
		data, err := entry.MarshalNDJSON()
		if err != nil {
			w.errorsCh <- err
			continue
		}
		if _, err := w.writer.Write(data); err != nil {
			w.errorsCh <- err
			continue
		}
	}
}

// Write adds an entry to the index
func (w *Writer) Write(entry *FileEntry) error {
	select {
	case w.entryCh <- entry:
		return nil
	case err := <-w.errorsCh:
		return err
	}
}

// Close flushes and closes the writer
func (w *Writer) Close() error {
	close(w.entryCh)
	w.wg.Wait()
	
	// Collect any remaining errors
	select {
	case err := <-w.errorsCh:
		return err
	default:
	}
	
	return w.writer.Flush()
}
