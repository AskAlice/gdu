package indexer

import (
	"bufio"
	"encoding/json"
	"io"
)

// Reader reads index entries from a file
type Reader struct {
	scanner *bufio.Scanner
}

// NewReader creates a new Reader
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// Next reads the next entry from the index
func (r *Reader) Next() (*FileEntry, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	var entry FileEntry
	if err := json.Unmarshal(r.scanner.Bytes(), &entry); err != nil {
		return nil, err
	}

	return &entry, nil
}
