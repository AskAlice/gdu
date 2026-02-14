package indexer

import (
	"encoding/json"
	"path/filepath"
	"time"
)

// FileEntry represents a file in the index
type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Mtime int64  `json:"mtime"` // Unix seconds
	Ctime int64  `json:"ctime"` // Unix seconds
	Ext   string `json:"ext"`
}

// NewFileEntry creates a new FileEntry
func NewFileEntry(name, path string, size int64, mtime, ctime time.Time) *FileEntry {
	ext := filepath.Ext(name)
	return &FileEntry{
		Name:  name,
		Path:  path,
		Size:  size,
		Mtime: mtime.Unix(),
		Ctime: ctime.Unix(),
		Ext:   ext,
	}
}

// GetMtime returns the modification time as time.Time
func (e *FileEntry) GetMtime() time.Time {
	if e.Mtime == 0 {
		return time.Time{}
	}
	return time.Unix(e.Mtime, 0)
}

// GetCtime returns the creation time as time.Time  
func (e *FileEntry) GetCtime() time.Time {
	if e.Ctime == 0 {
		return time.Time{}
	}
	return time.Unix(e.Ctime, 0)
}

// MarshalNDJSON marshals the entry as NDJSON (newline-delimited JSON)
func (e *FileEntry) MarshalNDJSON() ([]byte, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
