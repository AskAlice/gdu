# Integration: One Binary, Index-While-Crawl, Exit Prompt

## 1. One binary (gdu)

gdu is the only binary. Subcommands:
- `gdu [path]` — analyze + index while crawling
- `gdu search [-i index] [query]` — search index (CLI)
- `gdu serve [-i index] [-p port]` — run search web UI (HTTP + Unix socket)

The `cmd/gdu/subcommands.go` adds search and serve. In your `cmd/gdu/main.go` init():

```go
addSearchAndServeCommands(rootCmd)
```

## 2. Index storage (config directory)

Indexes are stored in the OS config directory, not inside the scanned directory:
- **Linux/macOS**: `~/.config/gdu/indexes/` (or `$XDG_CONFIG_HOME/gdu/indexes/`)
- **Windows**: `%APPDATA%\gdu\indexes\`

Filenames encode the absolute scanned path:
```
/home/alice/projects  ->  ~/.config/gdu/indexes/home-alice-projects.gds
/                     ->  ~/.config/gdu/indexes/root.gds
```

The `pkg/indexdir` package provides all path resolution functions.

## 3. Ancestor-aware index resolution

When gdu scans a path, it uses `indexdir.ResolveIndexAndScanRoot(absPath)` to avoid
creating redundant narrow indexes. The function walks up from the requested path to find
the deepest existing ancestor index:

```go
import "github.com/dundee/gdu/v5/pkg/indexdir"

// In the analyze startup:
scanArg := "."
if len(args) > 0 { scanArg = args[0] }
absPath, _ := filepath.Abs(scanArg)

// ResolveIndexAndScanRoot returns:
//   idxPath  - the index file to write (may be an ancestor's)
//   scanRoot - the directory to actually scan (may be wider than absPath)
idxPath, scanRoot, err := indexdir.ResolveIndexAndScanRoot(absPath)
```

Examples:
- First run: `gdu /home/alice/projects` → creates `home-alice-projects.gds`, scans `/home/alice/projects`
- Re-run same path: → overwrites `home-alice-projects.gds`, scans `/home/alice/projects`
- Subfolder run: `gdu /home/alice/projects/foo` → finds `home-alice-projects.gds`, re-scans `/home/alice/projects` (not just foo)
- Broader run: `gdu /home/alice` → no ancestor index → creates `home-alice.gds`, scans `/home/alice`

## 4. Index while crawling

In `pkg/analyze/parallel.go` in `processDir`, when each file is processed, write to the index:

```go
// Add to Analyzer:
indexWriter *indexer.Writer

// In processDir, for each file (after setPlatformSpecificAttrs):
if a.indexWriter != nil {
    ctime := GetCreationTime(entryPath, info)
    e := indexer.NewFileEntry(name, entryPath, info.Size(), info.ModTime(), ctime)
    a.indexWriter.Write(e)
}
```

At analysis start (in app):
```go
idxPath, scanRoot, _ := indexdir.ResolveIndexAndScanRoot(absPath)
// Use scanRoot as the analysis target (may be wider than what user typed)
idxFile, _ := os.Create(idxPath) // truncate: fresh index on every run
writer := indexer.NewWriter(idxFile)
// ... pass writer to analyzer, analyze scanRoot ...
// On completion:
writer.Close()
idxFile.Close()
```

## 5. Exit prompt (Escape / quit)

When the user hits Escape or tries to quit, show a terminal prompt instead of exiting.

In `tui/keys.go` or the quit handler:

```go
import "github.com/dundee/gdu/v5/internal/indexexit"

// On quit key (Escape / q):
// 1. Ensure index is saved (writer.Close())
// 2. Stop TUI (app.Stop() / screen.Fini())
// 3. Then:
if indexexit.OnExitPrompt(idxPath, 8765) {
    svc, _ := gdsserve.New(gdsserve.Config{IndexPath: idxPath, HTTPPort: 8765})
    svc.Start()  // blocks until Ctrl+C
} else {
    indexexit.PrintServeCommand(idxPath, 8765)
}
```

`PrintServeCommand` prints:
```
Index saved. To search later, run:
  gdu serve -i "/home/alice/.config/gdu/indexes/home-alice-projects.gds" -p 8765
```

## 6. Auto-resolution in search/serve

When `gdu search` or `gdu serve` are run without `-i`, they auto-detect the index:
1. Check for an index matching the current directory (`indexdir.FindIndexForDir(cwd)`)
2. Fall back to the most recently modified index (`indexdir.FindLatestIndex()`)

This means users can just run `gdu search *.go` after indexing without specifying a file.

## 7. Obsolete: cache files in scanned directory

The old approach of writing `.gdu-cache-*.gds` files inside the scanned directory
(via `indexer.FindCacheFile` / `indexer.CacheFileName`) is **replaced** by the config
directory approach. Those functions in `pkg/indexer/cache.go` are no longer used by the
subcommands and can be removed or kept for backward compatibility.
