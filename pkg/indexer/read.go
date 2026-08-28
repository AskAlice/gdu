package indexer

import (
	"bufio"
	"io"
)

// Reader streams FileEntry records from an NDJSON source.
type Reader struct {
	scanner *bufio.Scanner
}

// NewReader creates a streaming NDJSON reader.
func NewReader(r io.Reader) *Reader {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	return &Reader{scanner: s}
}

// Next advances to the next entry. Returns false at EOF or on error.
func (rd *Reader) Next() bool { return rd.scanner.Scan() }

// Entry parses and returns the current line as a FileEntry.
func (rd *Reader) Entry() (FileEntry, error) {
	return UnmarshalNDJSON(rd.scanner.Bytes())
}

// Err returns any scanner error.
func (rd *Reader) Err() error { return rd.scanner.Err() }
