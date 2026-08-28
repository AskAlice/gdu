package indexer

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/dundee/gdu/v5/internal/common"
)

var walkConcurrency = make(chan struct{}, 3*runtime.GOMAXPROCS(0))

// WalkStats holds aggregate statistics from a walk.
type WalkStats struct {
	TotalFiles int64
	TotalSize  int64
}

// Walker performs a parallel directory walk and streams FileEntry records.
type Walker struct {
	writer     *Writer
	ignoreDir  common.ShouldDirBeIgnored
	ignoreFile common.ShouldFileBeIgnored
	noHidden   bool
	noCross    bool
	rootDev    uint64

	Stats WalkStats
	wg    sync.WaitGroup
	bound *boundary
}

// NewWalker creates a Walker that sends entries to w.
func NewWalker(w *Writer, ignoreDir common.ShouldDirBeIgnored, ignoreFile common.ShouldFileBeIgnored, noHidden, noCross bool) *Walker {
	return &Walker{
		writer:     w,
		ignoreDir:  ignoreDir,
		ignoreFile: ignoreFile,
		noHidden:   noHidden,
		noCross:    noCross,
	}
}

// Walk scans root and all subdirectories, streaming FileEntry records to the writer.
// Returns aggregate stats. The caller must still call Writer.Close().
func (wk *Walker) Walk(root string) (WalkStats, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return wk.Stats, err
	}
	wk.bound = newBoundary(abs)
	if wk.noCross {
		wk.rootDev = deviceID(abs)
	}
	wk.processDir(abs)
	wk.wg.Wait()
	return wk.Stats, nil
}

func (wk *Walker) processDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, de := range entries {
		name := de.Name()
		path := filepath.Join(dir, name)

		if IsCacheName(name) {
			continue
		}
		if wk.noHidden && name[0] == '.' {
			continue
		}
		if wk.ignoreDir != nil && de.IsDir() && wk.ignoreDir(name, path) {
			continue
		}
		if wk.ignoreFile != nil && !de.IsDir() && wk.ignoreFile(name) {
			continue
		}

		info, err := de.Info()
		if err != nil {
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			tgt, _ := os.Readlink(path)
			wk.writer.Send(aliasEntry(path, tgt))
			continue
		}

		if de.IsDir() {
			if ok, tgt := wk.bound.decide(dir, path, false, ""); !ok {
				if tgt != "" {
					wk.writer.Send(aliasEntry(path, tgt))
				}
				continue
			}
			if wk.noCross && deviceID(path) != wk.rootDev {
				wk.writer.Send(aliasEntry(path, path))
				continue
			}
			wk.wg.Add(1)
			walkConcurrency <- struct{}{}
			go func(p string) {
				defer func() { <-walkConcurrency; wk.wg.Done() }()
				wk.processDir(p)
			}(path)
			continue
		}

		if !info.Mode().IsRegular() {
			continue
		}

		entry := NewFileEntry(path, info)
		wk.writer.Send(entry)
		atomic.AddInt64(&wk.Stats.TotalFiles, 1)
		atomic.AddInt64(&wk.Stats.TotalSize, info.Size())
	}
}
