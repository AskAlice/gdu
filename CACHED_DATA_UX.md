# Cached Data UX Enhancement

## Problem

Currently, gdu shows a loading screen every time it scans a directory, even if we already have indexed data from a previous run. This creates a poor UX for re-running gdu on the same paths.

## Proposed Solution

**Show cached index data immediately**, then optionally refresh in the background:

1. On startup, check if an index exists for the scan path
2. If it does, load and display it immediately in the TUI
3. Show cached entries in a **different color** (e.g., dimmed/gray) to indicate they're from cache
4. Optionally trigger a background scan to refresh the data
5. As new data comes in, update the display and change colors to normal

## Benefits

✅ **Instant feedback** - No more staring at loading screen  
✅ **Visual distinction** - Users know what's cached vs fresh  
✅ **Flexible** - Can choose to skip refresh if data is recent enough  
✅ **Better UX** - Especially for large directories that were just scanned

## Implementation Plan

### 1. App Initialization (cmd/gdu/app/app.go)

```go
import "github.com/dundee/gdu/v5/pkg/indexdir"

func (app *App) Run() error {
    absPath, _ := filepath.Abs(app.Path)
    idxPath, scanRoot, _ := indexdir.ResolveIndexAndScanRoot(absPath)
    
    // Check if index exists and is recent (optional: age check)
    indexExists := false
    if stat, err := os.Stat(idxPath); err == nil {
        indexExists = true
        ageMinutes := time.Since(stat.ModTime()).Minutes()
        app.Logger.Printf("Found index: %s (age: %.1f min)", idxPath, ageMinutes)
    }
    
    if indexExists && app.Flags.UseCachedData {
        // Load cached data first
        if err := app.LoadCachedIndex(idxPath); err == nil {
            app.ShowCachedDataInTUI()  // Display immediately with gray color
        }
    }
    
    // Then start the actual scan (updates TUI as it goes)
    return app.AnalyzeDirectory(scanRoot, idxPath)
}
```

### 2. TUI Color Scheme (tui/tui.go)

Add color distinction for cached vs fresh data:

```go
const (
    ColorFresh  = tcell.ColorWhite   // Normal color
    ColorCached = tcell.ColorGray    // Dimmed for cached data
)

type FileItem struct {
    // ... existing fields ...
    IsCached bool  // Mark if loaded from cache
}

func (ui *UI) RenderItem(item *FileItem) {
    color := ColorFresh
    if item.IsCached {
        color = ColorCached
    }
    // ... render with appropriate color ...
}
```

### 3. Index Loading (pkg/analyze/cached.go) - NEW FILE

```go
package analyze

import (
    "github.com/dundee/gdu/v5/pkg/indexer"
    "github.com/dundee/gdu/v5/pkg/device"
)

// LoadFromIndex loads a directory tree from an index file
func LoadFromIndex(indexPath string) (*device.Dir, error) {
    f, err := os.Open(indexPath)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    
    reader := indexer.NewReader(f)
    root := &device.Dir{
        Name: "/",
        Files: make([]*device.File, 0),
    }
    
    // Build directory tree from index entries
    for {
        entry, err := reader.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, err
        }
        
        // Add entry to tree structure
        addEntryToTree(root, entry)
    }
    
    return root, nil
}

func addEntryToTree(root *device.Dir, entry *indexer.FileEntry) {
    // Parse path and create parent directories if needed
    // Add file to appropriate directory
    // Mark as IsCached = true
}
```

### 4. Flag for Cached Data (cmd/gdu/app/flags.go)

Add a flag to control this behavior:

```go
type Flags struct {
    // ... existing flags ...
    UseCachedData   bool   // Use cached index data if available
    RefreshInBg     bool   // Refresh data in background after showing cache
    MaxCacheAge     int    // Don't use cache if older than N minutes (0 = always use)
}

// In init():
flags.BoolVar(&af.UseCachedData, "use-cache", true, "Show cached data immediately if available")
flags.BoolVar(&af.RefreshInBg, "refresh", true, "Refresh data in background after showing cache")
flags.IntVar(&af.MaxCacheAge, "max-cache-age", 60, "Max cache age in minutes (0 = always use)")
```

### 5. Update Integration Logic

Modify the analyzer to:
- Accept an optional "pre-loaded tree" from cache
- Mark new/updated entries as "fresh" (not cached)
- Update the TUI incrementally as scan progresses

```go
func (a *Analyzer) AnalyzeWithCachedData(cachedTree *device.Dir) error {
    // Display cached tree first
    a.UpdateTUI(cachedTree)
    
    // Start scan in background
    go func() {
        for dir := range a.dirChan {
            // Process directory
            // Update TUI with fresh data (changes color from gray to white)
            a.UpdateTUI(dir)
        }
    }()
    
    return a.scan()
}
```

## User Flow Examples

### Example 1: First Run (No Cache)
```bash
$ gdu ~/projects
[Normal loading screen]
[Shows results in white]
[Index saved to ~/.config/gdu/indexes/home-alice-projects.gds]
```

### Example 2: Re-run (With Cache)
```bash
$ gdu ~/projects
[Instantly shows cached data in gray]
[Background refresh starts]
[Colors change from gray → white as fresh data arrives]
[Updated index saved]
```

### Example 3: Skip Refresh
```bash
$ gdu ~/projects --no-refresh
[Instantly shows cached data in gray]
[No background scan]
[Much faster exit]
```

### Example 4: Cache Too Old
```bash
$ gdu ~/projects --max-cache-age 30
[Cache is 45 minutes old, skipped]
[Normal fresh scan]
```

## Visual Mockup

```
┌─────────────────────────────────────────────┐
│ /home/alice/projects [cached: 15min ago]   │
├─────────────────────────────────────────────┤
│ ⚡ node_modules       2.3 GB  (gray/cached) │
│ ⚡ .git               850 MB  (gray/cached) │
│ → src                120 MB  (white/fresh)  │ ← Just scanned
│ ⚡ dist                45 MB  (gray/cached) │
│ ⚡ package.json        12 KB  (gray/cached) │
└─────────────────────────────────────────────┘
```

Legend:
- ⚡ = Cached data (dimmed)
- → = Fresh data (normal color)

## Implementation Priority

**Phase 1: MVP** (Quick win)
1. Load cached index on startup
2. Display in TUI with gray color
3. Update colors as scan progresses

**Phase 2: Polish**
1. Add `--use-cache`, `--max-cache-age` flags
2. Show cache age in TUI header
3. Progress indicator for background refresh

**Phase 3: Advanced**
1. Smart diffing (show only changed files)
2. Incremental updates (only scan changed dirs)
3. Cache invalidation based on mtime

## Files to Modify

- [ ] `cmd/gdu/app/app.go` - Add cache loading logic
- [ ] `cmd/gdu/app/flags.go` - Add cache-related flags
- [ ] `pkg/analyze/cached.go` - NEW: Index loading and tree building
- [ ] `tui/tui.go` - Add color distinction for cached data
- [ ] `tui/show.go` - Update rendering to use colors
- [ ] `INTEGRATION.md` - Document new feature

## Testing

```go
func TestCachedDataDisplay(t *testing.T) {
    // 1. Run gdu, create index
    // 2. Re-run gdu with --use-cache
    // 3. Verify cached data shows immediately
    // 4. Verify colors change as scan progresses
}

func TestCacheAgeCheck(t *testing.T) {
    // 1. Create old index (>60 min)
    // 2. Run with --max-cache-age 30
    // 3. Verify cache is skipped
}
```

## Compatibility

- ✅ Backward compatible - cache is optional
- ✅ Works with existing indexes
- ✅ No breaking changes to index format
- ✅ Can be disabled with `--use-cache=false`

## Summary

This enhancement provides **instant feedback** by showing cached data while maintaining accuracy through background refresh. Users get the best of both worlds: speed AND freshness.
