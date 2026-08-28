package indexer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/dundee/gdu/v5/pkg/analyze"
)

// FileEntry represents a single file in the index
type FileEntry struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   int64  `json:"size,omitempty"`
	Mtime  int64  `json:"mtime,omitempty"`
	Ctime  int64  `json:"ctime,omitempty"`
	Ext    string `json:"ext,omitempty"`
	Type   string `json:"type,omitempty"`   // "alias" for symlink/mount; empty = file
	Target string `json:"target,omitempty"` // alias destination
	Gone   bool   `json:"gone,omitempty"`   // tombstone after delete
}

// NewFileEntry creates a FileEntry from file info and path
func NewFileEntry(path string, info os.FileInfo) FileEntry {
	var ctime int64
	ct := analyze.GetCreationTime(path, info)
	if !ct.IsZero() {
		ctime = ct.Unix()
	}
	return FileEntry{
		Name:  info.Name(),
		Path:  path,
		Size:  info.Size(),
		Mtime: info.ModTime().Unix(),
		Ctime: ctime,
		Ext:   filepath.Ext(info.Name()),
	}
}

// MarshalNDJSON returns a JSON line (no trailing newline)
func (e *FileEntry) MarshalNDJSON() ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalNDJSON parses a JSON line into a FileEntry
func UnmarshalNDJSON(line []byte) (FileEntry, error) {
	var e FileEntry
	err := json.Unmarshal(line, &e)
	return e, err
}

// GetMtime returns the modification time
func (e *FileEntry) GetMtime() time.Time {
	return time.Unix(e.Mtime, 0)
}

// GetCtime returns the creation time (zero if unavailable)
func (e *FileEntry) GetCtime() time.Time {
	if e.Ctime == 0 {
		return time.Time{}
	}
	return time.Unix(e.Ctime, 0)
}
