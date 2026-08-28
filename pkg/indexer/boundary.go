package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dundee/gdu/v5/pkg/device"
)

// IsCacheName reports whether name is a gdu index cache file.
func IsCacheName(name string) bool {
	return strings.HasPrefix(name, ".gdu-cache-") && strings.HasSuffix(name, ".ndjson")
}

type boundary struct {
	root   string
	mounts map[string]*device.Device
	mu     sync.Mutex
	seen   map[uint64]string
}

func newBoundary(root string) *boundary {
	b := &boundary{
		root:   filepath.Clean(root),
		mounts: map[string]*device.Device{},
		seen:   map[uint64]string{},
	}
	if mounts, err := device.Getter.GetMounts(); err == nil {
		for _, m := range mounts {
			b.mounts[filepath.Clean(m.MountPoint)] = m
		}
	}
	if id := deviceID(b.root); id != 0 {
		b.seen[id] = b.root
	}
	return b
}

func isUnder(path, root string) bool {
	path, root = filepath.Clean(path), filepath.Clean(root)
	if path == root {
		return true
	}
	sep := string(os.PathSeparator)
	if !strings.HasSuffix(root, sep) {
		root += sep
	}
	return strings.HasPrefix(path, root)
}

// decide whether to walk path. symlink/alias mounts never descend.
// A new-device mount descends only if it is under the walk root (subchild or root).
func (b *boundary) decide(parent, path string, symlink bool, linkTgt string) (descend bool, aliasTo string) {
	if symlink {
		return false, linkTgt
	}
	if IsCacheName(filepath.Base(path)) {
		return false, ""
	}
	pid, did := deviceID(parent), deviceID(path)
	mnt, listed := b.mounts[filepath.Clean(path)]
	crossed := did != 0 && pid != 0 && did != pid
	if !crossed && !listed {
		return true, ""
	}
	tgt := path
	if listed && mnt.Name != "" {
		tgt = mnt.Name
	}
	if !isUnder(path, b.root) {
		return false, tgt
	}
	if crossed {
		b.mu.Lock()
		first, ok := b.seen[did]
		if !ok {
			b.seen[did] = path
		}
		b.mu.Unlock()
		if ok {
			return false, first
		}
		return true, ""
	}
	// same-device bind/alias mount: record, do not walk again
	return false, tgt
}

func aliasEntry(path, target string) FileEntry {
	return FileEntry{
		Name:   filepath.Base(path),
		Path:   path,
		Type:   "alias",
		Target: target,
	}
}

// PathGone reports whether path is covered by a tombstone prefix.
func PathGone(path string, gone []string) bool {
	sep := string(os.PathSeparator)
	for _, g := range gone {
		if path == g || strings.HasPrefix(path, g+sep) {
			return true
		}
	}
	return false
}
