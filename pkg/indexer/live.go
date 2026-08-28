package indexer

import (
	"os"
	"path/filepath"

	"github.com/dundee/gdu/v5/pkg/fs"
)

// SaveFromTree writes an index of item (and children) to CachePath(root).
// Skips symlink/mount aliases so the cache does not double-count storage.
func SaveFromTree(root string, item fs.Item) error {
	if item == nil {
		return nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	cache := CachePath(abs)
	tmp := cache + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := NewWriter(f, 1024)
	dumpTree(w, item, "", newBoundary(abs))
	if err = w.Close(); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err = f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, cache)
}

func dumpTree(w *Writer, item fs.Item, parent string, b *boundary) {
	path := item.GetPath()
	if item.IsDir() {
		if parent != "" {
			if ok, tgt := b.decide(parent, path, false, ""); !ok {
				if tgt != "" {
					w.Send(aliasEntry(path, tgt))
				}
				return
			}
		}
		for c := range item.GetFiles(fs.SortByName, fs.SortAsc) {
			dumpTree(w, c, path, b)
		}
		return
	}
	if IsCacheName(item.GetName()) {
		return
	}
	if s, ok := item.(fs.SymlinkItem); ok && s.GetSymlinkTarget() != "" {
		w.Send(aliasEntry(path, s.GetSymlinkTarget()))
		return
	}
	if item.GetFlag() == '@' {
		w.Send(aliasEntry(path, ""))
		return
	}
	w.Send(FileEntry{
		Name:  item.GetName(),
		Path:  path,
		Size:  item.GetSize(),
		Mtime: item.GetMtime().Unix(),
		Ext:   filepath.Ext(item.GetName()),
	})
}

// RemovePath appends a tombstone so gds skips this path and its children.
func RemovePath(root, deleted string) error {
	if deleted == "" {
		return nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(CachePath(abs), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := (&FileEntry{Path: deleted, Gone: true}).MarshalNDJSON()
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

// LoadGone returns tombstone paths from an NDJSON index.
func LoadGone(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var gone []string
	rd := NewReader(f)
	for rd.Next() {
		e, err := rd.Entry()
		if err == nil && e.Gone && e.Path != "" {
			gone = append(gone, e.Path)
		}
	}
	return gone
}

// DirChanged is true when the on-disk listing no longer matches the in-memory dir.
func DirChanged(dir fs.Item) bool {
	if dir == nil {
		return false
	}
	entries, err := os.ReadDir(dir.GetPath())
	if err != nil {
		return false
	}
	known := map[string]struct{}{}
	for f := range dir.GetFilesLocked(fs.SortByName, fs.SortAsc) {
		known[f.GetName()] = struct{}{}
	}
	n := 0
	for _, e := range entries {
		if IsCacheName(e.Name()) {
			continue
		}
		if _, ok := known[e.Name()]; !ok {
			return true
		}
		n++
	}
	return n != len(known)
}
